// Copyright © 2019 Weald Technology Trading
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

package wallet

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/wealdtech/go-ecodec"
	hd "github.com/wealdtech/go-eth2-wallet-hd"
	nd "github.com/wealdtech/go-eth2-wallet-nd"
	types "github.com/wealdtech/go-eth2-wallet-types"
)

// walletOptions are the optons used when opening and creating wallets.
type walletOptions struct {
	store      types.Store
	encryptor  types.Encryptor
	walletType string
	passphrase []byte
}

// Option gives options to OpenWallet and CreateWallet.
type Option interface {
	apply(*walletOptions)
}

type optionFunc func(*walletOptions)

func (f optionFunc) apply(o *walletOptions) {
	f(o)
}

// WithStore sets the store for the wallet.
func WithStore(store types.Store) Option {
	return optionFunc(func(o *walletOptions) {
		o.store = store
	})
}

// WithEncryptor sets the encryptor for the wallet.
func WithEncryptor(encryptor types.Encryptor) Option {
	return optionFunc(func(o *walletOptions) {
		o.encryptor = encryptor
	})
}

// WithPassphrase sets the passphrase for the wallet.
func WithPassphrase(passphrase []byte) Option {
	return optionFunc(func(o *walletOptions) {
		o.passphrase = passphrase
	})
}

// WithType sets the type for the wallet.
func WithType(walletType string) Option {
	return optionFunc(func(o *walletOptions) {
		o.walletType = walletType
	})
}

// ImportWallet imports a wallet from its encrypted export.
func ImportWallet(encryptedData []byte, passphrase []byte) (types.Wallet, error) {
	type walletExt struct {
		Wallet *walletInfo `json:"wallet"`
	}

	data, err := ecodec.Decrypt(encryptedData, passphrase)
	if err != nil {
		return nil, err
	}

	ext := &walletExt{}
	err = json.Unmarshal(data, ext)
	if err != nil {
		return nil, err
	}

	var wallet types.Wallet
	switch ext.Wallet.Type {
	case "nd", "non-deterministic":
		wallet, err = nd.Import(encryptedData, passphrase, store, encryptor)
	case "hd", "hierarchical deterministic":
		wallet, err = hd.Import(encryptedData, passphrase, store, encryptor)
	default:
		return nil, fmt.Errorf("unsupported wallet type %q", ext.Wallet.Type)
	}
	return wallet, err
}

// OpenWallet opens an existing wallet.
// If the wallet does not exist an error is returned.
func OpenWallet(name string, opts ...Option) (types.Wallet, error) {
	options := walletOptions{
		store:     store,
		encryptor: encryptor,
	}
	for _, o := range opts {
		o.apply(&options)
	}

	data, err := store.RetrieveWallet(name)
	if err != nil {
		return nil, err
	}
	return walletFromBytes(data)
}

// CreateWallet creates a wallet.
// If the wallet already exists an error is returned.
func CreateWallet(name string, opts ...Option) (types.Wallet, error) {
	options := walletOptions{
		store:      store,
		encryptor:  encryptor,
		passphrase: nil,
		walletType: "nd",
	}
	for _, o := range opts {
		o.apply(&options)
	}

	switch options.walletType {
	case "nd", "non-deterministic":
		return nd.CreateWallet(name, options.store, options.encryptor)
	case "hd", "hierarchical deterministic":
		return hd.CreateWallet(name, options.passphrase, options.store, options.encryptor)
	default:
		return nil, fmt.Errorf("unhandled wallet type %q", options.walletType)
	}
}

type walletInfo struct {
	ID   uuid.UUID `json:"uuid"`
	Name string    `json:"name"`
	Type string    `json:"type"`
}

// Wallets provides information on the available wallets.
func Wallets() <-chan types.Wallet {
	ch := make(chan types.Wallet, 1024)
	go func() {
		for data := range store.RetrieveWallets() {
			wallet, err := walletFromBytes(data)
			if err == nil {
				ch <- wallet
			}
		}
		close(ch)
	}()
	return ch
}

func walletFromBytes(data []byte) (types.Wallet, error) {
	info := &walletInfo{}
	err := json.Unmarshal(data, info)
	if err != nil {
		return nil, err
	}
	var wallet types.Wallet
	switch info.Type {
	case "nd", "non-deterministic":
		wallet, err = nd.DeserializeWallet(data, store, encryptor)
	case "hd", "hierarchical deterministic":
		wallet, err = hd.DeserializeWallet(data, store, encryptor)
	default:
		return nil, fmt.Errorf("unsupported wallet type %q", info.Type)
	}
	return wallet, err
}
