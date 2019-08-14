/*
 * Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. ur file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use ur file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 *
 */

package issuer

import (
	"sync"

	"github.com/gardener/controller-manager-library/pkg/resources"

	api "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
)

func NewReferencedSecrets() *ReferencedSecrets {
	return &ReferencedSecrets{
		secretToIssuers: map[resources.ObjectName]resources.ObjectNameSet{},
		issuerToSecret:  map[resources.ObjectName]resources.ObjectName{},
	}
}

type ReferencedSecrets struct {
	lock            sync.Mutex
	secretToIssuers map[resources.ObjectName]resources.ObjectNameSet
	issuerToSecret  map[resources.ObjectName]resources.ObjectName
}

func newObjectName(namespace, name string) resources.ObjectName {
	if namespace == "" {
		namespace = "default"
	}
	return resources.NewObjectName(namespace, name)
}

func (rs *ReferencedSecrets) RememberIssuerSecret(issuer *api.Issuer) bool {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	secretRef := issuer.Spec.ACME.PrivateKeySecretRef
	issuerName := newObjectName(issuer.Namespace, issuer.Name)
	if secretRef == nil {
		return rs.removeIssuer(issuerName)
	}
	secretName := newObjectName(secretRef.Namespace, secretRef.Name)
	return rs.updateIssuerSecret(issuerName, secretName)
}

func (rs *ReferencedSecrets) RemoveIssuer(issuerName resources.ObjectName) bool {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.removeIssuer(issuerName)
}

func (rs *ReferencedSecrets) IssuerNamesFor(secretName resources.ObjectName) resources.ObjectNameSet {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	set, ok := rs.secretToIssuers[secretName]
	if !ok {
		return nil
	}
	return set.Copy()
}

func (rs *ReferencedSecrets) removeIssuer(issuerName resources.ObjectName) bool {
	secretName, ok := rs.issuerToSecret[issuerName]
	if ok {
		delete(rs.issuerToSecret, issuerName)
		rs.secretToIssuers[secretName].Remove(issuerName)
		if len(rs.secretToIssuers[secretName]) == 0 {
			delete(rs.secretToIssuers, secretName)
		}
	}
	return ok
}

func (rs *ReferencedSecrets) updateIssuerSecret(issuerName, secretName resources.ObjectName) bool {
	oldSecretName, ok := rs.issuerToSecret[issuerName]
	if ok && oldSecretName == secretName {
		return false
	}

	rs.removeIssuer(issuerName)

	rs.issuerToSecret[issuerName] = secretName
	set := rs.secretToIssuers[secretName]
	if set == nil {
		set = resources.ObjectNameSet{}
		rs.secretToIssuers[secretName] = set
	}
	set.Add(issuerName)

	return true
}
