package resolver

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WriteResolverConfig writes the resolver configuration to a ConfigMap or Secret
func WriteResolverConfig(ctx context.Context, c client.Client, namespace, name, key, configType, content string) error {
	if configType == "Secret" {
		return writeToSecret(ctx, c, namespace, name, key, content)
	}
	return writeToConfigMap(ctx, c, namespace, name, key, content)
}

// writeToConfigMap writes content to a ConfigMap
func writeToConfigMap(ctx context.Context, c client.Client, namespace, name, key, content string) error {
	cm := &corev1.ConfigMap{}
	err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cm)

	if err != nil {
		if errors.IsNotFound(err) {
			// Create new ConfigMap
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Data: map[string]string{
					key: content,
				},
			}
			if err := c.Create(ctx, cm); err != nil {
				return fmt.Errorf("failed to create ConfigMap: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Update existing ConfigMap
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[key] = content

	if err := c.Update(ctx, cm); err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	return nil
}

// writeToSecret writes content to a Secret
func writeToSecret(ctx context.Context, c client.Client, namespace, name, key, content string) error {
	secret := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, secret)

	if err != nil {
		if errors.IsNotFound(err) {
			// Create new Secret
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				StringData: map[string]string{
					key: content,
				},
			}
			if err := c.Create(ctx, secret); err != nil {
				return fmt.Errorf("failed to create Secret: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get Secret: %w", err)
	}

	// Update existing Secret
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data[key] = []byte(content)

	if err := c.Update(ctx, secret); err != nil {
		return fmt.Errorf("failed to update Secret: %w", err)
	}

	return nil
}
