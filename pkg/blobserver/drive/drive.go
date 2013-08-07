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
	"net/http"
	"time"

	"camlistore.org/pkg/blobserver"
	"camlistore.org/pkg/blobserver/drive/service"
	"camlistore.org/pkg/jsonconfig"
	"camlistore.org/third_party/code.google.com/p/goauth2/oauth"
)

const (
	GoogleOAuth2AuthURL  = "https://accounts.google.com/o/oauth2/auth"
	GoogleOAuth2TokenURL = "https://accounts.google.com/o/oauth2/token"
)

type DriveStorage struct {
	*blobserver.SimpleBlobHubPartitionMap
	service *driveservice.DriveService
}

func newFromConfig(_ blobserver.Loader, config jsonconfig.Obj) (storage blobserver.Storage, err error) {
	auth := config.RequiredObject("auth")
	oauthConf := &oauth.Config{
		ClientId:     auth.RequiredString("client_id"),
		ClientSecret: auth.RequiredString("client_secret"),
		Scope:        "",
		AuthURL:      GoogleOAuth2AuthURL,
		TokenURL:     GoogleOAuth2TokenURL,
	}

	// force refreshes the access token on start,
	transport := &oauth.Transport{
		Token: &oauth.Token{
			AccessToken:  "",
			RefreshToken: auth.RequiredString("refresh_token"),
			Expiry:       time.Now(),
		},
		Config:    oauthConf,
		Transport: http.DefaultTransport,
	}

	service, err := driveservice.New(transport, config.RequiredString("parent_id"))
	sto := &DriveStorage{
		SimpleBlobHubPartitionMap: &blobserver.SimpleBlobHubPartitionMap{},
		service:                   service,
	}
	return sto, err
}

func init() {
	blobserver.RegisterStorageConstructor("drive", blobserver.StorageConstructor(newFromConfig))
}
