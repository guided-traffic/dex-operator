/*
Copyright 2025 Guided Traffic GmbH.

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

package builder

import (
	"context"
	"fmt"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// buildStorage converts a DexStorageSpec into a Dex StorageConfig.
// Credentials are extracted into envs and TLS files are added to mounts.
func buildStorage(
	ctx context.Context,
	spec dexv1.DexStorageSpec,
	sr SecretResolver,
	namespace string,
	envs map[string][]byte,
) (StorageConfig, []MountedSecret, error) {
	sc := StorageConfig{Type: string(spec.Type)}
	var mounts []MountedSecret

	switch spec.Type {
	case dexv1.StorageKubernetes:
		sc.Config = map[string]any{"inCluster": true}

	case dexv1.StorageMemory:
		// no sub-config required

	case dexv1.StorageSQLite3:
		if spec.SQLite3 != nil {
			sc.Config = map[string]any{"file": spec.SQLite3.File}
		}

	case dexv1.StoragePostgres:
		if spec.Postgres != nil {
			cfg, pgMounts, err := buildPostgresConfig(ctx, spec.Postgres, sr, namespace, envs)
			if err != nil {
				return StorageConfig{}, nil, fmt.Errorf("postgres storage: %w", err)
			}
			sc.Config = cfg
			mounts = append(mounts, pgMounts...)
		}

	case dexv1.StorageEtcd:
		if spec.Etcd != nil {
			cfg, etcdMounts, err := buildEtcdConfig(ctx, spec.Etcd, sr, namespace)
			if err != nil {
				return StorageConfig{}, nil, fmt.Errorf("etcd storage: %w", err)
			}
			sc.Config = cfg
			mounts = append(mounts, etcdMounts...)
		}

	case dexv1.StorageMySQL:
		if spec.MySQL != nil {
			cfg, err := buildMySQLConfig(ctx, spec.MySQL, sr, namespace, envs)
			if err != nil {
				return StorageConfig{}, nil, fmt.Errorf("mysql storage: %w", err)
			}
			sc.Config = cfg
		}
	}

	return sc, mounts, nil
}

func buildPostgresConfig(
	ctx context.Context,
	s *dexv1.DexPostgresStorageSpec,
	sr SecretResolver,
	namespace string,
	envs map[string][]byte,
) (map[string]any, []MountedSecret, error) {
	cfg := map[string]any{
		"host":     s.Host,
		"database": s.Database,
		"user":     s.User,
	}

	var mounts []MountedSecret

	if s.PasswordRef != nil {
		envKey := storageEnvKey("POSTGRES_PASSWORD")
		ref, err := resolveEnvSecret(ctx, namespace, *s.PasswordRef, envKey, sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("postgres password: %w", err)
		}
		cfg["password"] = ref
	}

	setOptionalInt(cfg, "maxOpenConns", s.MaxOpenConns)
	setOptionalInt(cfg, "maxIdleConns", s.MaxIdleConns)
	setOptionalInt(cfg, "connMaxLifetime", s.ConnMaxLifetime)
	setOptionalInt(cfg, "connectionTimeout", s.ConnectionTimeout)

	if s.SSL != nil {
		sslCfg, sslMounts := buildPostgresSSL(s.SSL, namespace)
		cfg["ssl"] = sslCfg
		mounts = append(mounts, sslMounts...)
	}

	return cfg, mounts, nil
}

func buildPostgresSSL(s *dexv1.DexPostgresSSLSpec, namespace string) (map[string]any, []MountedSecret) {
	cfg := map[string]any{}
	var mounts []MountedSecret

	if s.Mode != "" {
		cfg["mode"] = s.Mode
	}

	if s.CARef != nil {
		path := "/etc/dex/certs/postgres-ca.pem"
		mounts = append(mounts, MountedSecret{
			Namespace:  namespace,
			SecretName: s.CARef.Name,
			SecretKey:  s.CARef.Key,
			MountPath:  path,
		})
		cfg["caFile"] = path
	}

	if s.CertRef != nil {
		path := "/etc/dex/certs/postgres-client-cert.pem"
		mounts = append(mounts, MountedSecret{
			Namespace:  namespace,
			SecretName: s.CertRef.Name,
			SecretKey:  s.CertRef.Key,
			MountPath:  path,
		})
		cfg["certFile"] = path
	}

	if s.KeyRef != nil {
		path := "/etc/dex/certs/postgres-client-key.pem"
		mounts = append(mounts, MountedSecret{
			Namespace:  namespace,
			SecretName: s.KeyRef.Name,
			SecretKey:  s.KeyRef.Key,
			MountPath:  path,
		})
		cfg["keyFile"] = path
	}

	return cfg, mounts
}

func buildEtcdConfig(
	ctx context.Context,
	s *dexv1.DexEtcdStorageSpec,
	sr SecretResolver,
	namespace string,
) (map[string]any, []MountedSecret, error) {
	cfg := map[string]any{
		"endpoints": s.Endpoints,
	}

	if s.Namespace != "" {
		cfg["namespace"] = s.Namespace
	}

	if s.Username != "" {
		cfg["username"] = s.Username
	}

	if s.PasswordRef != nil {
		val, err := resolveSecret(ctx, namespace, *s.PasswordRef, sr)
		if err != nil {
			return nil, nil, fmt.Errorf("etcd password: %w", err)
		}
		cfg["password"] = val
	}

	var mounts []MountedSecret
	if s.SSL != nil {
		sslCfg, sslMounts := buildEtcdSSL(s.SSL, namespace)
		cfg["ssl"] = sslCfg
		mounts = append(mounts, sslMounts...)
	}

	return cfg, mounts, nil
}

func buildEtcdSSL(s *dexv1.DexEtcdSSLSpec, namespace string) (map[string]any, []MountedSecret) {
	cfg := map[string]any{}
	var mounts []MountedSecret

	if s.ServerName != "" {
		cfg["serverName"] = s.ServerName
	}

	if s.CARef != nil {
		path := "/etc/dex/certs/etcd-ca.pem"
		mounts = append(mounts, MountedSecret{
			Namespace:  namespace,
			SecretName: s.CARef.Name,
			SecretKey:  s.CARef.Key,
			MountPath:  path,
		})
		cfg["caFile"] = path
	}

	if s.CertRef != nil {
		path := "/etc/dex/certs/etcd-client-cert.pem"
		mounts = append(mounts, MountedSecret{
			Namespace:  namespace,
			SecretName: s.CertRef.Name,
			SecretKey:  s.CertRef.Key,
			MountPath:  path,
		})
		cfg["certFile"] = path
	}

	if s.KeyRef != nil {
		path := "/etc/dex/certs/etcd-client-key.pem"
		mounts = append(mounts, MountedSecret{
			Namespace:  namespace,
			SecretName: s.KeyRef.Name,
			SecretKey:  s.KeyRef.Key,
			MountPath:  path,
		})
		cfg["keyFile"] = path
	}

	return cfg, mounts
}

func buildMySQLConfig(
	ctx context.Context,
	s *dexv1.DexMySQLStorageSpec,
	sr SecretResolver,
	namespace string,
	envs map[string][]byte,
) (map[string]any, error) {
	cfg := map[string]any{}

	switch {
	case s.DSNRef != nil:
		envKey := storageEnvKey("MYSQL_DSN")
		ref, err := resolveEnvSecret(ctx, namespace, *s.DSNRef, envKey, sr, envs)
		if err != nil {
			return nil, fmt.Errorf("mysql DSN: %w", err)
		}
		cfg["dsn"] = ref

	case s.DSN != "":
		cfg["dsn"] = s.DSN
	}

	setOptionalInt(cfg, "maxOpenConns", s.MaxOpenConns)
	setOptionalInt(cfg, "maxIdleConns", s.MaxIdleConns)
	setOptionalInt(cfg, "connMaxLifetime", s.ConnMaxLifetime)

	return cfg, nil
}

// setOptionalInt sets key in cfg only when ptr is non-nil.
func setOptionalInt(cfg map[string]any, key string, ptr *int) {
	if ptr != nil {
		cfg[key] = *ptr
	}
}
