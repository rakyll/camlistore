// +build linux darwin

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

package fs

import (
	"os"

	"camlistore.org/pkg/blob"
	"camlistore.org/third_party/code.google.com/p/rsc/fuse"
)

type blobDir struct {
	fs *CamliFileSystem
}

func (b blobDir) Attr() fuse.Attr {
	return fuse.Attr{
		Mode: os.ModeDir | 0500,
		Uid:  uint32(os.Getuid()),
		Gid:  uint32(os.Getgid()),
	}
}

func (b *blobDir) ReadDir(intr fuse.Intr) ([]fuse.Dirent, fuse.Error) {
	// disable listing
	return []fuse.Dirent{}, nil
}

func (b *blobDir) Lookup(name string, intr fuse.Intr) (fuse.Node, fuse.Error) {
	if ref, ok := blob.Parse(name); ok {
		return &node{fs: b.fs, blobref: ref}, nil
	}
	return nil, fuse.ENOENT
}
