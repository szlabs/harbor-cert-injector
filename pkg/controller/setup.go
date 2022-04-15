// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"reflect"
	"sync"

	"github.com/go-logr/logr"

	ctrl "sigs.k8s.io/controller-runtime"
)

var controllers sync.Map

// Controller for doing reconcile.
type Controller interface {
	// SetupWithManager registers reconcile controller to manager.
	SetupWithManager(mgr ctrl.Manager) error
}

// AddToControllerList register controllers.
func AddToControllerList(c Controller) {
	if c != nil {
		k := reflect.TypeOf(c).String()
		if _, ok := controllers.Load(k); !ok {
			controllers.Store(k, c)
		}
	}
}

// SetupControllers sets up all the registered controllers.
func SetupControllers(mgr ctrl.Manager, logger logr.Logger) error {
	var err error

	controllers.Range(func(k, v interface{}) bool {
		if c, ok := v.(Controller); ok {
			if err = c.SetupWithManager(mgr); err != nil {
				return false
			}

			logger.Info("Set up with controller manager", "controller", k)
		}

		return true
	})

	return err
}
