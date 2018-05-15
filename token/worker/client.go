//
// Copyright (c) SAS Institute Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package worker

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/sassoftware/relic/config"
	"github.com/sassoftware/relic/internal/workerrpc"
	"github.com/sassoftware/relic/token"
)

const (
	defaultRetries = 5
	defaultTimeout = 60 * time.Second
)

func (t *WorkerToken) request(path string, rr workerrpc.Request) (*workerrpc.Response, error) {
	req := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: t.addr, Path: path},
		Header: http.Header{"Auth-Cookie": []string{t.cookie}},
	}
	blob, err := json.Marshal(rr)
	if err != nil {
		return nil, err
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader(blob)), nil
	}
	return t.doRetry(req)
}

func (t *WorkerToken) Ping() error {
	_, err := t.request(workerrpc.Ping, workerrpc.Request{})
	return err
}

func (t *WorkerToken) Config() *config.TokenConfig {
	return t.tconf
}

func (t *WorkerToken) GetKey(keyName string) (token.Key, error) {
	kconf, err := t.config.GetKey(keyName)
	if err != nil {
		return nil, err
	}
	res, err := t.request(workerrpc.GetKey, workerrpc.Request{KeyName: kconf.Name()})
	if err != nil {
		return nil, err
	}
	pub, err := x509.ParsePKIXPublicKey(res.Value)
	if err != nil {
		return nil, err
	}
	return &workerKey{
		token:  t,
		kconf:  kconf,
		public: pub,
		id:     res.ID,
	}, nil
}

func (t *WorkerToken) Import(keyName string, privKey crypto.PrivateKey) (token.Key, error) {
	return nil, errors.New("function not implemented for worker token")
}

func (t *WorkerToken) ImportCertificate(cert *x509.Certificate, labelBase string) error {
	return errors.New("function not implemented for worker token")
}

func (t *WorkerToken) Generate(keyName string, keyType token.KeyType, bits uint) (token.Key, error) {
	return nil, errors.New("function not implemented for worker token")
}

func (t *WorkerToken) ListKeys(opts token.ListOptions) error {
	return errors.New("function not implemented for worker token")
}

type workerKey struct {
	token  *WorkerToken
	kconf  *config.KeyConfig
	public crypto.PublicKey
	id     []byte
}

func (k *workerKey) Config() *config.KeyConfig {
	return k.kconf
}

func (k *workerKey) Public() crypto.PublicKey {
	return k.public
}

func (k *workerKey) GetID() []byte {
	return k.id
}

func (k *workerKey) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	rr := workerrpc.Request{
		KeyName: k.kconf.Name(),
		Digest:  digest,
		Hash:    uint(opts.HashFunc()),
	}
	if o, ok := opts.(*rsa.PSSOptions); ok {
		rr.SaltLength = &o.SaltLength
	}
	res, err := k.token.request(workerrpc.Sign, rr)
	if err != nil {
		return nil, err
	}
	return res.Value, nil
}

func (k *workerKey) ImportCertificate(cert *x509.Certificate) error {
	return errors.New("function not implemented for worker token")
}
