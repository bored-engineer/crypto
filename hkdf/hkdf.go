// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package hkdf implements the HMAC-based Extract-and-Expand Key Derivation
// Function (HKDF) as defined in RFC 5869.
//
// HKDF is a cryptographic key derivation function (KDF) with the goal of
// expanding limited input keying material into one or more cryptographically
// strong secret keys.
package hkdf // import "github.com/bored-engineer/crypto/hkdf"

import (
	"crypto/hmac"
	"errors"
	"hash"
	"io"
)

// Extract generates a pseudorandom key for use with Expand from an input secret
// and an optional independent salt.
//
// Only use this function if you need to reuse the extracted key with multiple
// Expand invocations and different context values. Most common scenarios,
// including the generation of multiple keys, should use New instead.
func Extract(hash func() hash.Hash, secret, salt []byte) []byte {
	if salt == nil {
		salt = make([]byte, hash().Size())
	}
	extractor := hmac.New(hash, salt)
	extractor.Write(secret)
	return extractor.Sum(nil)
}

type hkdf struct {
	expander hash.Hash
	size     int

	info    []byte
	counter byte

	prev []byte
	buf  []byte
}

func (f *hkdf) Read(p []byte) (int, error) {
	// Check whether enough data can be generated
	need := len(p)
	remains := len(f.buf) + int(255-f.counter+1)*f.size
	if remains < need {
		return 0, errors.New("hkdf: entropy limit reached")
	}
	// Read any leftover from the buffer
	n := copy(p, f.buf)
	p = p[n:]

	// Fill the rest of the buffer
	for len(p) > 0 {
		f.expander.Reset()
		f.expander.Write(f.prev)
		f.expander.Write(f.info)
		f.expander.Write([]byte{f.counter})
		f.prev = f.expander.Sum(f.prev[:0])
		f.counter++

		// Copy the new batch into p
		f.buf = f.prev
		n = copy(p, f.buf)
		p = p[n:]
	}
	// Save leftovers for next run
	f.buf = f.buf[n:]

	return need, nil
}

// Expand returns a Reader, from which keys can be read, using the given
// pseudorandom key and optional context info, skipping the extraction step.
//
// The pseudorandomKey should have been generated by Extract, or be a uniformly
// random or pseudorandom cryptographically strong key. See RFC 5869, Section
// 3.3. Most common scenarios will want to use New instead.
func Expand(hash func() hash.Hash, pseudorandomKey, info []byte) io.Reader {
	expander := hmac.New(hash, pseudorandomKey)
	return &hkdf{expander, expander.Size(), info, 1, nil, nil}
}

// New returns a Reader, from which keys can be read, using the given hash,
// secret, salt and context info. Salt and info can be nil.
func New(hash func() hash.Hash, secret, salt, info []byte) io.Reader {
	prk := Extract(hash, secret, salt)
	return Expand(hash, prk, info)
}
