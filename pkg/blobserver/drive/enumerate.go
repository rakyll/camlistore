/*
Copyright 2013 Google Inc.

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

package drive

import (
	"time"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/blobserver"
)

var _ blobserver.MaxEnumerateConfig = (*DriveStorage)(nil)

func (sto *DriveStorage) MaxEnumerate() int { return 1000 }

func (sto *DriveStorage) EnumerateBlobs(dest chan<- blob.SizedRef, after string, limit int, wait time.Duration) error {
	defer close(dest)

	// TODO: fix pagination
	files, err := sto.service.List(after, limit)
	if err != nil {
		return err
	}
	for _, f := range files {
		b, ok := blob.Parse(f.Title)
		if !ok {
			continue
		}
		dest <- blob.SizedRef{Ref: b, Size: f.FileSize}
	}

	return nil
}
