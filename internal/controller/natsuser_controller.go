/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	natsv1alpha1 "github.com/jradikk/nats-auth-operator/api/v1alpha1"
	jwtpkg "github.com/jradikk/nats-auth-operator/internal/jwt"
	"github.com/jradikk/nats-auth-operator/internal/token"
)

const (
	natsUserFinalizer = "nats.jradikk/user-finalizer"
)

// NatsUserReconciler reconciles a NatsUser object
type NatsUserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nats.jradikk,resources=natsusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nats.jradikk,resources=natsusers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nats.jradikk,resources=natsusers/finalizers,verbs=update
// +kubebuilder:rbac:groups=nats.jradikk,resources=natsauthconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=nats.jradikk,resources=natsaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *NatsUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the NatsUser instance
	user := &natsv1alpha1.NatsUser{}
	if err := r.Get(ctx, req.NamespacedName, user); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !user.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, user)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(user, natsUserFinalizer) {
		controllerutil.AddFinalizer(user, natsUserFinalizer)
		if err := r.Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Get the referenced NatsAuthConfig
	authConfig, err := r.getAuthConfig(ctx, user)
	if err != nil {
		log.Error(err, "Failed to get NatsAuthConfig")
		r.updateStatus(user, natsv1alpha1.UserStateError, err.Error())
		if err := r.Status().Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Determine auth type
	authType := user.Spec.AuthType
	if authType == natsv1alpha1.UserAuthTypeInherit {
		authType = natsv1alpha1.UserAuthType(authConfig.Spec.Mode)
	}

	// Reconcile based on auth type
	var reconcileErr error
	switch authType {
	case natsv1alpha1.UserAuthTypeJWT:
		reconcileErr = r.reconcileJWTUser(ctx, user, authConfig)
	case natsv1alpha1.UserAuthTypeToken:
		reconcileErr = r.reconcileTokenUser(ctx, user, authConfig)
	default:
		reconcileErr = fmt.Errorf("unsupported auth type: %s", authType)
	}

	// Update status
	now := metav1.Now()
	user.Status.LastReconciled = &now
	user.Status.ObservedGeneration = user.Generation

	if reconcileErr != nil {
		log.Error(reconcileErr, "Failed to reconcile user")
		r.updateStatus(user, natsv1alpha1.UserStateError, reconcileErr.Error())
		r.updateCondition(user, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "ReconcileError",
			Message: reconcileErr.Error(),
		})
		if err := r.Status().Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Minute}, reconcileErr
	}

	r.updateStatus(user, natsv1alpha1.UserStateReady, "User reconciled successfully")
	r.updateCondition(user, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "ReconcileSuccess",
		Message: "NatsUser reconciled successfully",
	})

	if err := r.Status().Update(ctx, user); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("NatsUser reconciled successfully", "authType", authType)

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *NatsUserReconciler) reconcileJWTUser(ctx context.Context, user *natsv1alpha1.NatsUser, authConfig *natsv1alpha1.NatsAuthConfig) error {
	log := log.FromContext(ctx)

	// Validate that account reference is provided
	if user.Spec.AccountRef == nil {
		return fmt.Errorf("accountRef is required for JWT mode")
	}

	// Get the referenced NatsAccount
	account, err := r.getAccount(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to get NatsAccount: %w", err)
	}

	// Wait for account to be ready
	if account.Status.AccountID == "" {
		return fmt.Errorf("NatsAccount is not ready yet")
	}

	// Check if user credentials secret already exists
	secretName := fmt.Sprintf("%s-user-creds", user.Name)
	existingSecret := &corev1.Secret{}
	checkErr := r.Get(ctx, client.ObjectKey{Namespace: user.Namespace, Name: secretName}, existingSecret)
	if checkErr == nil {
		// Credentials already exist - check if we need to update them
		if user.Status.PublicKey != "" && len(existingSecret.Data["user.creds"]) > 0 {
			// Credentials exist and status is set - no need to regenerate
			log.Info("User credentials already exist, skipping regeneration", "publicKey", user.Status.PublicKey)
			return nil
		}
	} else if !errors.IsNotFound(checkErr) {
		return fmt.Errorf("failed to check credentials secret: %w", checkErr)
	}

	// Get or create user seed
	userSeed, err := r.getOrCreateUserSeed(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to get user seed: %w", err)
	}

	// Create user manager
	userMgr, err := jwtpkg.NewUserManager(userSeed)
	if err != nil {
		return fmt.Errorf("failed to create user manager: %w", err)
	}

	// Get user public key
	userPubKey, err := userMgr.GetPublicKey()
	if err != nil {
		return fmt.Errorf("failed to get user public key: %w", err)
	}

	// Create user claims
	userName := user.Name
	if user.Spec.Username != "" {
		userName = user.Spec.Username
	}

	userClaims, err := userMgr.CreateUserClaims(userName, user.Spec.Permissions)
	if err != nil {
		return fmt.Errorf("failed to create user claims: %w", err)
	}

	// Get account keypair to sign the user JWT
	accountSeed, err := r.getAccountSeed(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to get account seed: %w", err)
	}

	accountMgr, err := jwtpkg.NewAccountManager(accountSeed)
	if err != nil {
		return fmt.Errorf("failed to create account manager: %w", err)
	}

	// Sign the user JWT
	userJWT, err := accountMgr.SignUserJWT(userClaims)
	if err != nil {
		return fmt.Errorf("failed to sign user JWT: %w", err)
	}

	// Generate credentials file
	credsContent := jwtpkg.GenerateCredsFile(userJWT, userSeed)

	// Store user credentials in a secret (secretName already declared above)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: user.Namespace,
		},
		StringData: map[string]string{
			"user.creds": credsContent,
			"user.jwt":   userJWT,
			"NATS_URL":   authConfig.Spec.NatsURL,
		},
		Data: map[string][]byte{
			"seed.nk": userSeed,
		},
	}

	if err := controllerutil.SetControllerReference(user, secret, r.Scheme); err != nil {
		return err
	}

	// Create or update the secret (reuse existingSecret from above)
	existingSecret = &corev1.Secret{}
	checkErr = r.Get(ctx, client.ObjectKey{Namespace: user.Namespace, Name: secretName}, existingSecret)
	if checkErr != nil {
		if errors.IsNotFound(checkErr) {
			if err := r.Create(ctx, secret); err != nil {
				return fmt.Errorf("failed to create credentials secret: %w", err)
			}
		} else {
			return checkErr
		}
	} else {
		existingSecret.StringData = secret.StringData
		existingSecret.Data = secret.Data
		if err := r.Update(ctx, existingSecret); err != nil {
			return fmt.Errorf("failed to update credentials secret: %w", err)
		}
	}

	// Update status
	user.Status.PublicKey = userPubKey
	user.Status.SecretRef = natsv1alpha1.SecretRef{
		Name:      secretName,
		Namespace: user.Namespace,
	}

	return nil
}

func (r *NatsUserReconciler) reconcileTokenUser(ctx context.Context, user *natsv1alpha1.NatsUser, authConfig *natsv1alpha1.NatsAuthConfig) error {
	// Determine username
	username := user.Spec.Username
	if username == "" {
		// Generate username
		var err error
		username, err = token.GenerateUsername(user.Name)
		if err != nil {
			return fmt.Errorf("failed to generate username: %w", err)
		}
	}

	// Determine password
	var password string
	if user.Spec.PasswordFrom != nil {
		if user.Spec.PasswordFrom.Generate {
			// Generate password
			var err error
			password, err = token.GeneratePassword()
			if err != nil {
				return fmt.Errorf("failed to generate password: %w", err)
			}
		} else if user.Spec.PasswordFrom.SecretRef != nil {
			// Get password from secret
			secret := &corev1.Secret{}
			key := client.ObjectKey{
				Namespace: user.Spec.PasswordFrom.SecretRef.Namespace,
				Name:      user.Spec.PasswordFrom.SecretRef.Name,
			}
			if err := r.Get(ctx, key, secret); err != nil {
				return fmt.Errorf("failed to get password secret: %w", err)
			}
			password = string(secret.Data["password"])
		}
	} else {
		// Default: generate password
		var err error
		password, err = token.GeneratePassword()
		if err != nil {
			return fmt.Errorf("failed to generate password: %w", err)
		}
	}

	// Store user credentials in a secret
	secretName := fmt.Sprintf("%s-user-creds", user.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: user.Namespace,
		},
		StringData: map[string]string{
			"USERNAME":  username,
			"PASSWORD":  password,
			"NATS_URL":  authConfig.Spec.NatsURL,
		},
	}

	if err := controllerutil.SetControllerReference(user, secret, r.Scheme); err != nil {
		return err
	}

	// Create or update the secret
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Namespace: user.Namespace, Name: secretName}, existingSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, secret); err != nil {
				return fmt.Errorf("failed to create credentials secret: %w", err)
			}
		} else {
			return err
		}
	} else {
		// Only update if password/username changed
		if existingSecret.Data == nil ||
			string(existingSecret.Data["USERNAME"]) != username ||
			string(existingSecret.Data["PASSWORD"]) != password {
			existingSecret.StringData = secret.StringData
			if err := r.Update(ctx, existingSecret); err != nil {
				return fmt.Errorf("failed to update credentials secret: %w", err)
			}
		}
	}

	// Update status
	user.Status.SecretRef = natsv1alpha1.SecretRef{
		Name:      secretName,
		Namespace: user.Namespace,
	}

	return nil
}

func (r *NatsUserReconciler) getOrCreateUserSeed(ctx context.Context, user *natsv1alpha1.NatsUser) ([]byte, error) {
	// Check if existing seed is specified
	if user.Spec.ExistingSeedSecret != nil {
		secret := &corev1.Secret{}
		key := client.ObjectKey{
			Namespace: user.Spec.ExistingSeedSecret.Namespace,
			Name:      user.Spec.ExistingSeedSecret.Name,
		}
		if err := r.Get(ctx, key, secret); err != nil {
			return nil, fmt.Errorf("failed to get user seed secret: %w", err)
		}

		seed, ok := secret.Data["user.seed"]
		if !ok {
			seed, ok = secret.Data["seed.nk"]
			if !ok {
				return nil, fmt.Errorf("user seed not found in secret")
			}
		}

		return seed, nil
	}

	// Create new user keypair
	userMgr, err := jwtpkg.NewUserManager(nil)
	if err != nil {
		return nil, err
	}

	seed, err := userMgr.GetSeed()
	if err != nil {
		return nil, err
	}

	return seed, nil
}

func (r *NatsUserReconciler) getAuthConfig(ctx context.Context, user *natsv1alpha1.NatsUser) (*natsv1alpha1.NatsAuthConfig, error) {
	authConfig := &natsv1alpha1.NatsAuthConfig{}
	namespace := user.Spec.AuthConfigRef.Namespace
	if namespace == "" {
		namespace = user.Namespace
	}

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      user.Spec.AuthConfigRef.Name,
	}

	if err := r.Get(ctx, key, authConfig); err != nil {
		return nil, err
	}

	return authConfig, nil
}

func (r *NatsUserReconciler) getAccount(ctx context.Context, user *natsv1alpha1.NatsUser) (*natsv1alpha1.NatsAccount, error) {
	if user.Spec.AccountRef == nil {
		return nil, fmt.Errorf("accountRef is nil")
	}

	account := &natsv1alpha1.NatsAccount{}
	namespace := user.Spec.AccountRef.Namespace
	if namespace == "" {
		namespace = user.Namespace
	}

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      user.Spec.AccountRef.Name,
	}

	if err := r.Get(ctx, key, account); err != nil {
		return nil, err
	}

	return account, nil
}

func (r *NatsUserReconciler) getAccountSeed(ctx context.Context, account *natsv1alpha1.NatsAccount) ([]byte, error) {
	if account.Status.JWTSecretRef.Name == "" {
		return nil, fmt.Errorf("account JWT secret not ready")
	}

	secret := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: account.Status.JWTSecretRef.Namespace,
		Name:      account.Status.JWTSecretRef.Name,
	}

	if err := r.Get(ctx, key, secret); err != nil {
		return nil, fmt.Errorf("failed to get account JWT secret: %w", err)
	}

	seed, ok := secret.Data["account.seed"]
	if !ok {
		return nil, fmt.Errorf("account seed not found in secret")
	}

	return seed, nil
}

func (r *NatsUserReconciler) handleDeletion(ctx context.Context, user *natsv1alpha1.NatsUser) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(user, natsUserFinalizer) {
		// Cleanup logic here if needed
		controllerutil.RemoveFinalizer(user, natsUserFinalizer)
		if err := r.Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *NatsUserReconciler) updateStatus(user *natsv1alpha1.NatsUser, state natsv1alpha1.UserState, reason string) {
	user.Status.State = state
	user.Status.Reason = reason
}

func (r *NatsUserReconciler) updateCondition(user *natsv1alpha1.NatsUser, condition metav1.Condition) {
	condition.LastTransitionTime = metav1.Now()
	found := false
	for i, c := range user.Status.Conditions {
		if c.Type == condition.Type {
			user.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		user.Status.Conditions = append(user.Status.Conditions, condition)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NatsUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&natsv1alpha1.NatsUser{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
