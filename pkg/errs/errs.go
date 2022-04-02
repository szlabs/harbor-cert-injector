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

package errs

import (
	"errors"
	"fmt"
)

// TLSNotEnabledError ...
var TLSNotEnabledError = New("No need to inject CA as TLS is not enabled")

// New error
func New(message string) error {
	return fmt.Errorf("error: %s", message)
}

// Errorf return an error with a specified format.
func Errorf(format string, fields ...interface{}) error {
	return fmt.Errorf(format, fields...)
}

// Wrap error with extra message.
func Wrap(message string, err error) error {
	return fmt.Errorf("%w:%s", err, message)
}

// IsTLSNotEnabledError checks if the error is TLSNotEnabledError.
func IsTLSNotEnabledError(err error) bool {
	return errors.Is(err, TLSNotEnabledError)
}
