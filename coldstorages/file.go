package coldstorages

import (
	"context"
	"encoding/base64"
	"encoding/gob"
	"os"
	"sync"

	"github.com/JGpGH/golfu/element"
	"github.com/JGpGH/golfu/errors"
	"github.com/JGpGH/golfu/lockmap"
)

type FileStorage[T element.Indexable] struct {
	basePath string
	lockMap  *lockmap.LockMap
}

type resource struct {
	refCount int
	lock     *sync.RWMutex
}

func NewFileStorage[T element.Indexable](basePath string) *FileStorage[T] {
	os.MkdirAll(basePath, os.ModePerm)
	return &FileStorage[T]{
		basePath: basePath,
		lockMap:  lockmap.New(),
	}
}

func (fs *FileStorage[T]) path_from_index(index string) string {
	return fs.basePath + "/" + base64.URLEncoding.EncodeToString([]byte(index))
}

func (fs *FileStorage[T]) writeToFile(element T) error {
	index := element.Index()
	unlock := fs.lockMap.Lock(index)
	defer unlock()
	if index == "" {
		return errors.ErrInvalidIndex
	}
	path := fs.path_from_index(index)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	return encoder.Encode(element)
}

func (fs *FileStorage[T]) read(index string, ref *T) error {
	unlock := fs.lockMap.Rlock(index)
	defer unlock()
	if index == "" {
		return errors.ErrInvalidIndex
	}
	path := fs.path_from_index(index)
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(ref); err != nil {
		return err
	}

	return nil
}

func (fs *FileStorage[T]) Set(ctx context.Context, values []T) error {
	for _, element := range values {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := fs.writeToFile(element); err != nil {
			return err
		}
	}
	return nil
}

func (fs *FileStorage[T]) OnEviction(ctx context.Context, values []T) error {
	return nil
}

func (fs *FileStorage[T]) Get(ctx context.Context, index string) (*T, error) {
	result := make(chan *T, 1)
	errChan := make(chan error, 1)

	go func() {
		var element T
		unlock := fs.lockMap.Rlock(index)
		err := fs.read(index, &element)
		unlock()
		if err == nil {
			result <- &element
			return
		}
		if os.IsNotExist(err) {
			err = errors.ErrNotFound
		}
		errChan <- err
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errChan:
		return nil, err
	case res := <-result:
		return res, nil
	}
}

func (fs *FileStorage[T]) Gets(ctx context.Context, indexes []string) (map[string]T, error) {
	result := make(map[string]T)
	var err error
	for _, index := range indexes {
		res, geterr := fs.Get(ctx, index)
		if geterr == nil {
			result[index] = *res
		} else if err != errors.ErrNotFound {
			return result, geterr
		} else {
			err = geterr
		}
	}
	return result, err
}

func (fs *FileStorage[T]) Delete(ctx context.Context, indexes []string) error {
	for _, index := range indexes {
		unlock := fs.lockMap.Lock(index)
		path := fs.path_from_index(index)
		removeErr := os.Remove(path)
		unlock()
		if removeErr != nil && !os.IsNotExist(removeErr) {
			return removeErr
		}
	}
	return nil
}
