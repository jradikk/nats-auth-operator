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

	"github.com/nats-io/nkeys"
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
)

const (
	natsAccountFinalizer = "nats.jradikk/account-finalizer"
)

// NatsAccountReconciler reconciles a NatsAccount object
type NatsAccountReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nats.jradikk,resources=natsaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nats.jradikk,resources=natsaccounts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nats.jradikk,resources=natsaccounts/finalizers,verbs=update
// +kubebuilder:rbac:groups=nats.jradikk,resources=natsauthconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *NatsAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the NatsAccount instance
	account := &natsv1alpha1.NatsAccount{}
	if err := r.Get(ctx, req.NamespacedName, account); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !account.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, account)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(account, natsAccountFinalizer) {
		controllerutil.AddFinalizer(account, natsAccountFinalizer)
		if err := r.Update(ctx, account); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Get the referenced NatsAuthConfig
	authConfig, err := r.getAuthConfig(ctx, account)
	if err != nil {
		log.Error(err, "Failed to get NatsAuthConfig")
		r.updateCondition(account, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "AuthConfigNotFound",
			Message: err.Error(),
		})
		if err := r.Status().Update(ctx, account); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Validate that AuthConfig is in JWT mode
	if authConfig.Spec.Mode != natsv1alpha1.AuthModeJWT && authConfig.Spec.Mode != natsv1alpha1.AuthModeMixed {
		err := fmt.Errorf("NatsAuthConfig must be in JWT or mixed mode for NatsAccount")
		log.Error(err, "Invalid auth mode")
		r.updateCondition(account, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "InvalidAuthMode",
			Message: err.Error(),
		})
		if err := r.Status().Update(ctx, account); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Reconcile the account
	if err := r.reconcileAccount(ctx, account, authConfig); err != nil {
		log.Error(err, "Failed to reconcile account")
		r.updateCondition(account, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "ReconcileError",
			Message: err.Error(),
		})
		if err := r.Status().Update(ctx, account); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Update status
	now := metav1.Now()
	account.Status.LastReconciled = &now
	account.Status.ObservedGeneration = account.Generation

	r.updateCondition(account, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "ReconcileSuccess",
		Message: "NatsAccount reconciled successfully",
	})

	if err := r.Status().Update(ctx, account); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("NatsAccount reconciled successfully", "accountID", account.Status.AccountID)

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *NatsAccountReconciler) reconcileAccount(ctx context.Context, account *natsv1alpha1.NatsAccount, authConfig *natsv1alpha1.NatsAuthConfig) error {
	log := log.FromContext(ctx)

	// Check if JWT secret already exists
	jwtSecretName := fmt.Sprintf("%s-account-jwt", account.Name)
	existingSecret := &corev1.Secret{}
	jwtSecretExists := false
	err := r.Get(ctx, client.ObjectKey{Namespace: account.Namespace, Name: jwtSecretName}, existingSecret)
	if err == nil {
		jwtSecretExists = true
		// JWT already exists - verify it matches the status before skipping
		if account.Status.AccountID != "" && len(existingSecret.Data["account.jwt"]) > 0 && len(existingSecret.Data["account.seed"]) > 0 {
			// Verify the seed in the secret generates the same account ID as in status
			seedData := existingSecret.Data["account.seed"]
			kp, err := nkeys.FromSeed(seedData)
			if err == nil {
				pubKey, err := kp.PublicKey()
				if err == nil && pubKey == account.Status.AccountID {
					// JWT exists, status matches seed - no need to regenerate
					log.Info("Account JWT already exists and matches status, skipping regeneration", "accountID", account.Status.AccountID)
					return nil
				}
			}
			// If we get here, the status doesn't match the seed - need to regenerate
			log.Info("Account ID in status doesn't match seed, will regenerate", "statusID", account.Status.AccountID)
		}
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check JWT secret: %w", err)
	}

	// Get or create account seed
	accountSeed, err := r.getOrCreateAccountSeed(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to get account seed: %w", err)
	}

	// Create account manager
	accountMgr, err := jwtpkg.NewAccountManager(accountSeed)
	if err != nil {
		return fmt.Errorf("failed to create account manager: %w", err)
	}

	// Get account public key
	accountPubKey, err := accountMgr.GetPublicKey()
	if err != nil {
		return fmt.Errorf("failed to get account public key: %w", err)
	}

	// Create account claims
	accountClaims, err := accountMgr.CreateAccountClaims(
		account.Name,
		account.Spec.Description,
		account.Spec.Limits,
	)
	if err != nil {
		return fmt.Errorf("failed to create account claims: %w", err)
	}

	// Get operator keypair to sign the account JWT
	operatorSeed, err := r.getOperatorSeed(ctx, authConfig)
	if err != nil {
		return fmt.Errorf("failed to get operator seed: %w", err)
	}

	operatorMgr, err := jwtpkg.NewOperatorManager(operatorSeed, "")
	if err != nil {
		return fmt.Errorf("failed to create operator manager: %w", err)
	}

	// Sign the account JWT
	accountJWT, err := operatorMgr.SignAccountJWT(accountClaims)
	if err != nil {
		return fmt.Errorf("failed to sign account JWT: %w", err)
	}

	// Store account JWT in a secret
	jwtSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jwtSecretName,
			Namespace: account.Namespace,
		},
		Data: map[string][]byte{
			"account.jwt":  []byte(accountJWT),
			"account.seed": accountSeed,
		},
	}

	if err := controllerutil.SetControllerReference(account, jwtSecret, r.Scheme); err != nil {
		return err
	}

	// Create or update the secret
	if !jwtSecretExists {
		if err := r.Create(ctx, jwtSecret); err != nil {
			return fmt.Errorf("failed to create JWT secret: %w", err)
		}
		log.Info("Created new account JWT secret", "secret", jwtSecretName)
	} else {
		existingSecret.Data = jwtSecret.Data
		if err := r.Update(ctx, existingSecret); err != nil {
			return fmt.Errorf("failed to update JWT secret: %w", err)
		}
		log.Info("Updated account JWT secret", "secret", jwtSecretName)
	}

	// Update status first (so the NatsAuthConfig controller can find it)
	account.Status.AccountID = accountPubKey
	account.Status.PublicKey = accountPubKey
	account.Status.JWTSecretRef = natsv1alpha1.SecretRef{
		Name:      jwtSecretName,
		Namespace: account.Namespace,
	}

	// Trigger NatsAuthConfig reconciliation to update resolver_preload
	// Only do this once when we first create/update the JWT
	if err := r.triggerAuthConfigReconcile(ctx, authConfig); err != nil {
		return fmt.Errorf("failed to trigger auth config reconciliation: %w", err)
	}

	return nil
}

func (r *NatsAccountReconciler) getOrCreateAccountSeed(ctx context.Context, account *natsv1alpha1.NatsAccount) ([]byte, error) {
	// Check if existing seed is specified
	if account.Spec.ExistingSeedSecret != nil {
		secret := &corev1.Secret{}
		key := client.ObjectKey{
			Namespace: account.Spec.ExistingSeedSecret.Namespace,
			Name:      account.Spec.ExistingSeedSecret.Name,
		}
		if err := r.Get(ctx, key, secret); err != nil {
			return nil, fmt.Errorf("failed to get account seed secret: %w", err)
		}

		seed, ok := secret.Data["account.seed"]
		if !ok {
			return nil, fmt.Errorf("account seed not found in secret")
		}

		return seed, nil
	}

	// Create new account and store the seed
	accountMgr, err := jwtpkg.NewAccountManager(nil)
	if err != nil {
		return nil, err
	}

	seed, err := accountMgr.GetSeed()
	if err != nil {
		return nil, err
	}

	return seed, nil
}

func (r *NatsAccountReconciler) getAuthConfig(ctx context.Context, account *natsv1alpha1.NatsAccount) (*natsv1alpha1.NatsAuthConfig, error) {
	authConfig := &natsv1alpha1.NatsAuthConfig{}
	namespace := account.Spec.AuthConfigRef.Namespace
	if namespace == "" {
		namespace = account.Namespace
	}

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      account.Spec.AuthConfigRef.Name,
	}

	if err := r.Get(ctx, key, authConfig); err != nil {
		return nil, err
	}

	return authConfig, nil
}

func (r *NatsAccountReconciler) getOperatorSeed(ctx context.Context, authConfig *natsv1alpha1.NatsAuthConfig) ([]byte, error) {
	var secretName, secretNamespace, seedKey string

	if authConfig.Spec.JWT.OperatorSeedSecret != nil {
		secretName = authConfig.Spec.JWT.OperatorSeedSecret.Name
		secretNamespace = authConfig.Spec.JWT.OperatorSeedSecret.Namespace
		seedKey = authConfig.Spec.JWT.OperatorSeedSecret.Key
		if seedKey == "" {
			seedKey = "operator.seed"
		}
	} else {
		secretName = fmt.Sprintf("%s-operator-seed", authConfig.Name)
		secretNamespace = authConfig.Namespace
		seedKey = "operator.seed"
	}

	secret := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: secretNamespace,
		Name:      secretName,
	}

	if err := r.Get(ctx, key, secret); err != nil {
		return nil, fmt.Errorf("failed to get operator seed secret: %w", err)
	}

	seed, ok := secret.Data[seedKey]
	if !ok {
		return nil, fmt.Errorf("operator seed key %q not found in secret", seedKey)
	}

	return seed, nil
}

// triggerAuthConfigReconcile forces a reconciliation of the NatsAuthConfig
// This is needed when accounts are created/updated/deleted to refresh resolver_preload
func (r *NatsAccountReconciler) triggerAuthConfigReconcile(ctx context.Context, authConfig *natsv1alpha1.NatsAuthConfig) error {
	log := log.FromContext(ctx)

	// Update a dummy annotation to trigger reconciliation
	if authConfig.Annotations == nil {
		authConfig.Annotations = make(map[string]string)
	}
	authConfig.Annotations["nats.jradikk/last-account-update"] = time.Now().Format(time.RFC3339)

	if err := r.Update(ctx, authConfig); err != nil {
		return fmt.Errorf("failed to trigger auth config reconciliation: %w", err)
	}

	log.Info("Triggered NatsAuthConfig reconciliation", "authConfig", authConfig.Name)
	return nil
}

func (r *NatsAccountReconciler) handleDeletion(ctx context.Context, account *natsv1alpha1.NatsAccount) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(account, natsAccountFinalizer) {
		// Trigger NatsAuthConfig reconciliation to remove this account from resolver_preload
		authConfig, err := r.getAuthConfig(ctx, account)
		if err == nil {
			// Trigger reconciliation to update resolver_preload without this account
			_ = r.triggerAuthConfigReconcile(ctx, authConfig)
		}

		controllerutil.RemoveFinalizer(account, natsAccountFinalizer)
		if err := r.Update(ctx, account); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *NatsAccountReconciler) updateCondition(account *natsv1alpha1.NatsAccount, condition metav1.Condition) {
	condition.LastTransitionTime = metav1.Now()
	found := false
	for i, c := range account.Status.Conditions {
		if c.Type == condition.Type {
			account.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		account.Status.Conditions = append(account.Status.Conditions, condition)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NatsAccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&natsv1alpha1.NatsAccount{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
