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
	"strings"

	"camlistore.org/third_party/code.google.com/p/goauth2/oauth"
	driveclient "camlistore.org/third_party/code.google.com/p/google-api-go-client/drive/v2"
)

const (
	MimeTypeRegistrationMetadata = "application/vnd.camlistore.metadata"
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

func (s *DriveService) GetRegistrations(
	after string, limit int) (paths []string, next string, err error) {
	return nil, "", nil
}

func (s *DriveService) RegisterBlob(m []string) (err error) {
	// lock for m[0]
	// defer unlock m[0]
	prop, err := s.Get(m[0])
	if err != nil || prop == nil {
		prop = &driveclient.File{Title: m[0], Description: m[1] + ";"}
		prop.MimeType = MimeTypeRegistrationMetadata
		prop.Parents = []*driveclient.ParentReference{&driveclient.ParentReference{Id: s.parentId}}
		_, err = s.apiservice.Files.Insert(prop).Do()
		return
	}
	// check prop value contains m[1], else append
	if strings.Contains(prop.Description, m[1]) {
		return
	}
	prop.Description += m[1] + ";"
	_, err = s.apiservice.Files.Update(prop.Id, prop).Do()
	return
}

func (s *DriveService) UnregisterBlob(m []string) error {
	// TODO: implement
	return nil
}

func (s *DriveService) Get(id string) (*driveclient.File, error) {
	req := s.apiservice.Files.List()
	// TODO: use field selectors
	query := fmt.Sprintf("'%s' in parents and title = '%s' and trashed = false", s.parentId, id)
	req.Q(query)
	files, err := req.Do()

	if err != nil || len(files.Items) < 1 {
		return nil, err
	}
	return files.Items[0], err
}

// Lists at most limitted number of files
// from the parent folder
func (s *DriveService) List(
	pageToken string, limit int) (files []*driveclient.File, err error) {
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

func (s *DriveService) Upsert(
	id string, parents []string, blob io.Reader) (file *driveclient.File, err error) {
	if file, err = s.Get(id); err != nil {
		return
	}

	if file == nil {
		// Register file so we can be aware of its existence during enumaration
		if err = s.RegisterBlob(parents); err != nil {
			return
		}

		file = &driveclient.File{Title: id}
		file.Parents = []*driveclient.ParentReference{&driveclient.ParentReference{Id: s.parentId}}
		return s.apiservice.Files.Insert(file).Media(blob).Do()
	}
	// TODO: handle large blobs
	return s.apiservice.Files.Update(file.Id, file).Media(blob).Do()
}

func (s *DriveService) Fetch(id string) (body io.ReadCloser, size int64, err error) {
	file, err := s.Get(id)

	// TODO: maybe in the case of no download link, remove the file
	// file should have been corrupted
	if file == nil || file.DownloadUrl != "" {
		return
	}

	req, _ := http.NewRequest("GET", file.DownloadUrl, nil)
	resp, err := s.transport.RoundTrip(req)
	if err != nil {
		return
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
