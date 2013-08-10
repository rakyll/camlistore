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

package driveservice

import (
	"fmt"
	"io"
	"net/http"

	"camlistore.org/third_party/code.google.com/p/goauth2/oauth"
	driveclient "camlistore.org/third_party/code.google.com/p/google-api-go-client/drive/v2"
)

type DriveService struct {
	transport  *oauth.Transport
	apiservice *driveclient.Service
	parentId   string
}

func New(transport *oauth.Transport, parentId string) (*DriveService, error) {
	apiservice, err := driveclient.New(transport.Client())
	if err != nil {
		return nil, err
	}
	service := &DriveService{transport: transport, apiservice: apiservice, parentId: parentId}
	return service, err
}

func (s *DriveService) MakeDirs(names []string) (file *driveclient.File, err error) {
	parentId := s.parentId
	for _, name := range names {
		// lock the operation
		req := s.apiservice.Files.List()
		req.Q(fmt.Sprintf("title = '%s' and '%s' in parents", name, parentId))

		var files *driveclient.FileList
		if files, err = req.Do(); err != nil {
			return
		}

		if len(files.Items) < 1 {
			// create a new shard, if doesnt exist
			shard := &driveclient.File{Title: name}
			shard.MimeType = "application/vnd.google-apps.folder"
			shard.Parents = []*driveclient.ParentReference{&driveclient.ParentReference{Id: parentId}}
			if file, err = s.apiservice.Files.Insert(shard).Do(); err != nil {
				return
			}
		} else {
			file = files.Items[0]
		}
		parentId = file.Id
	}
	return
}

func (s *DriveService) Get(id string) (*driveclient.File, error) {
	req := s.apiservice.Files.List()
	// TODO: use field selectors
	// TODO: investigate ways to avoid this query
	query := fmt.Sprintf("'%s' in parents and title = '%s'", s.parentId, id)
	req.Q(query)
	files, err := req.Do()

	if err != nil || len(files.Items) < 1 {
		return nil, err
	}
	return files.Items[0], err
}

// Lists at most limitted number of files
// from the parent folder
func (s *DriveService) List(pageToken string, limit int) (files []*driveclient.File, err error) {
	req := s.apiservice.Files.List()
	req.Q(fmt.Sprintf("'%s' in parents", s.parentId))

	if pageToken != "" {
		req.PageToken(pageToken)
	}

	if limit > 0 {
		req.MaxResults(int64(limit))
	}

	result, err := req.Do()
	if err != nil {
		return
	}
	return result.Items, err
}

func (s *DriveService) Upsert(parents []string, id string, data io.Reader) (file *driveclient.File, err error) {
	if file, err = s.Get(id); err != nil {
		return
	}
	// TODO: file level lock is required once requests
	// wont be run serially
	if file == nil {
		// make a shard first
		var shard *driveclient.File
		if shard, err = s.MakeDirs(parents); err != nil {
			return
		}

		file = &driveclient.File{Title: id}
		// keep parent reference to the root parentId
		// so stat and get queries will be cheaper
		file.Parents = []*driveclient.ParentReference{
			&driveclient.ParentReference{Id: s.parentId},
			&driveclient.ParentReference{Id: shard.Id},
		}
		return s.apiservice.Files.Insert(file).Media(data).Do()
	}

	// TODO: handle large blobs
	return s.apiservice.Files.Update(file.Id, file).Media(data).Do()
}

func (s *DriveService) Fetch(id string) (io.ReadCloser, int64, error) {
	file, err := s.Get(id)

	// TODO: maybe in the case of no download link, remove the file
	if file == nil || file.DownloadUrl != "" {
		return nil, 0, err
	}

	req, _ := http.NewRequest("GET", file.DownloadUrl, nil)
	resp, err := s.transport.RoundTrip(req)
	if err != nil {
		return nil, 0, err
	}
	return resp.Body, file.FileSize, err
}

func (s *DriveService) Stat(id string) (int64, error) {
	file, err := s.Get(id)
	if err != nil || file == nil {
		return 0, err
	}
	return file.FileSize, err
}

func (s *DriveService) Remove(id string) error {
	// TODO: should trash or permanently remove?
	file, err := s.Get(id)
	if err == nil && file != nil {
		_, err = s.apiservice.Files.Trash(file.Id).Do()
	}
	return err
}
