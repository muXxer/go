package database

import (
    "github.com/dgraph-io/badger"
    "os"
    "path/filepath"
    "sync"
)

var databasesByName = make(map[string]*databaseImpl)

type databaseImpl struct {
    db       *badger.DB
    name     string
    openLock sync.Mutex
}

func Get(name string) Database {
    if database, exists := databasesByName[name]; exists {
        return database
    }

    databasesByName[name] = &databaseImpl{
        db:   nil,
        name: name,
    }

    return databasesByName[name]
}

func (this *databaseImpl) Open() error {
    this.openLock.Lock()
    defer this.openLock.Unlock()

    if this.db == nil {
        if _, err := os.Stat(DIRECTORY.GetValue()); os.IsNotExist(err) {
            if err := os.Mkdir(DIRECTORY.GetValue(), 0700); err != nil {
                return err
            }
        }

        opts := badger.DefaultOptions
        opts.Dir = DIRECTORY.GetValue() + string(filepath.Separator) + this.name
        opts.ValueDir = opts.Dir
        opts.Logger = &logger{}
        opts.Truncate = true

        db, err := badger.Open(opts)
        if err != nil {
            return err
        }
        this.db = db
    }

    return nil
}

func (this *databaseImpl) Set(key []byte, value []byte) error {
    if err := this.Open(); err != nil {
        return err
    }

    if err := this.db.Update(func(txn *badger.Txn) error { return txn.Set(key, value) }); err != nil {
        return err
    }

    return nil
}

func (this *databaseImpl) Get(key []byte) ([]byte, error) {
    var result []byte = nil
    var err error = nil

    if err = this.Open(); err == nil {
        err = this.db.View(func(txn *badger.Txn) error {
            item, err := txn.Get(key)
            if err != nil {
                return err
            }

            return item.Value(func(val []byte) error {
                result = append([]byte{}, val...)

                return nil
            })
        })
    }

    return result, err
}

func (this *databaseImpl) Close() error {
    this.openLock.Lock()
    defer this.openLock.Unlock()

    if this.db != nil {
        err := this.db.Close()

        this.db = nil

        if err != nil {
            return err
        }
    }

    return nil
}
