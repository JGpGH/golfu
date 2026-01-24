package coldstorages

import (
	"context"
	"encoding/base64"
	"encoding/gob"
	"os"

	"github.com/JGpGH/golfu/element"
	"github.com/JGpGH/golfu/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type FileStorage[T element.Indexable] struct {
	basePath string
}

func NewFileStorage[T element.Indexable](basePath string) *FileStorage[T] {
	os.MkdirAll(basePath, os.ModePerm)
	return &FileStorage[T]{
		basePath: basePath,
	}
}

func (fs *FileStorage[T]) path_from_index(index string) string {
	return fs.basePath + "/" + base64.URLEncoding.EncodeToString([]byte(index))
}

func (fs *FileStorage[T]) writeToFile(element T) error {
	index := element.Index()
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
	ctx, span := otel.GetTracerProvider().Tracer("FileStorage").Start(ctx, "FileStorage.Set")
	defer span.End()
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

func (fs *FileStorage[T]) Trash(ctx context.Context, values []T) error {
	return nil
}

func (fs *FileStorage[T]) Get(ctx context.Context, index string) (*T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("FileStorage").Start(ctx, "FileStorage.Get")
	defer span.End()
	span.SetAttributes(attribute.String("index", index))
	if ctx.Err() != nil {
		span.RecordError(ctx.Err())
		return nil, ctx.Err()
	}
	var element T
	err := fs.read(index, &element)
	if err == nil {
		return &element, nil
	}
	if os.IsNotExist(err) {
		err = errors.ErrNotFound
	} else {
		span.RecordError(err)
	}
	return nil, err
}

func (fs *FileStorage[T]) Gets(ctx context.Context, indexes []string) (map[string]T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("FileStorage").Start(ctx, "FileStorage.Gets")
	defer span.End()
	result := make(map[string]T)
	var err error
	for _, index := range indexes {
		if ctx.Err() != nil {
			span.RecordError(ctx.Err())
			return nil, ctx.Err()
		}
		var element T
		readerr := fs.read(index, &element)
		if readerr == nil {
			result[index] = element
		} else if os.IsNotExist(readerr) {
		} else {
			span.RecordError(err)
			return nil, err
		}
	}
	return result, nil
}
