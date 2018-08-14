// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package buckets

import (
	"bytes"
	"context"
	"time"

	monkit "gopkg.in/spacemonkeygo/monkit.v2"
	"storj.io/storj/pkg/paths"
	"storj.io/storj/pkg/storage/meta"
	"storj.io/storj/pkg/storage/objects"
)

var (
	mon = monkit.Package()
)

// Store creates an interface for interacting with buckets
type Store interface {
	Get(ctx context.Context, bucket string) (meta Meta, err error)
	Put(ctx context.Context, bucket string) (meta Meta, err error)
	Delete(ctx context.Context, bucket string) (err error)
	List(ctx context.Context, startAfter, endBefore string, limit int) (
		items []ListItem, more bool, err error)
	GetObjectStore(ctx context.Context, bucketName string) (store objects.Store, err error)
}

// ListItem is a single item in a listing
type ListItem struct {
	Bucket string
	Meta   Meta
}

// BucketStore contains objects store
type BucketStore struct {
	o objects.Store
}

// Meta is the bucket metadata struct
type Meta struct {
	Created time.Time
}

// NewStore instantiates BucketStore
func NewStore(obj objects.Store) Store {
	return &BucketStore{o: obj}
}

// GetObjectStore returns an implementation of objects.Store
func (b *BucketStore) GetObjectStore(ctx context.Context, bucket string) (objects.Store, error) {
	_, err := b.Get(ctx, bucket)
	if err != nil {
		// TODO(nat): When pull #216 is ready, we have to check
		// if the error is ErrKeyNotFound, and if so, return
		// minio.BucketNotFound error instead. Otherwise, S3
		// clients will only receive a "500 Internal Error"
		// message.
		return nil, err
	}
	prefixed := prefixedObjStore{
		o:      b.o,
		prefix: bucket,
	}
	return &prefixed, nil
}

// Get calls objects store Get
func (b *BucketStore) Get(ctx context.Context, bucket string) (meta Meta, err error) {
	defer mon.Task()(&ctx)(&err)
	p := paths.New(bucket)
	objMeta, err := b.o.Meta(ctx, p)
	if err != nil {
		return Meta{}, err
	}
	return convertMeta(objMeta), nil
}

// Put calls objects store Put
func (b *BucketStore) Put(ctx context.Context, bucket string) (meta Meta, err error) {
	defer mon.Task()(&ctx)(&err)
	p := paths.New(bucket)
	r := bytes.NewReader(nil)
	var exp time.Time
	m, err := b.o.Put(ctx, p, r, objects.SerializableMeta{}, exp)
	if err != nil {
		return Meta{}, err
	}
	return convertMeta(m), nil
}

// Delete calls objects store Delete
func (b *BucketStore) Delete(ctx context.Context, bucket string) (err error) {
	defer mon.Task()(&ctx)(&err)
	p := paths.New(bucket)
	return b.o.Delete(ctx, p)
}

// List calls objects store List
func (b *BucketStore) List(ctx context.Context, startAfter, endBefore string, limit int) (
	items []ListItem, more bool, err error) {
	defer mon.Task()(&ctx)(&err)
	objItems, more, err := b.o.List(ctx, nil, paths.New(startAfter), paths.New(endBefore), false, limit, meta.Modified)
	items = make([]ListItem, len(objItems))
	for i, itm := range objItems {
		items[i] = ListItem{
			Bucket: itm.Path.String(),
			Meta:   convertMeta(itm.Meta),
		}
	}
	return items, more, nil
}

// convertMeta converts stream metadata to object metadata
func convertMeta(m objects.Meta) Meta {
	return Meta{
		Created: m.Modified,
	}
}
