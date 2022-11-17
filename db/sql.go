package db

import (
	"errors"
	"github.com/gohumble/cashu-feni/cashu"
	"github.com/gohumble/cashu-feni/lightning"
	"github.com/gohumble/cashu-feni/lightning/lnbits"
	cashuLog "github.com/gohumble/cashu-feni/log"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"os"
	"path"
)

type SqlDatabase struct {
	db *gorm.DB
}

func NewSqlDatabase() MintStorage {
	if Config.Database.Sqlite != nil {
		return createSqliteDatabase()
	}
	return nil
}

func createSqliteDatabase() MintStorage {
	filePath := Config.Database.Sqlite.Path
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	db := SqlDatabase{db: open(sqlite.Open(path.Join(filePath, Config.Database.Sqlite.FileName)))}

	return db
}

func open(dialector gorm.Dialector) *gorm.DB {
	orm, err := gorm.Open(dialector,
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, FullSaveAssociations: true})
	if err != nil {
		panic(err)
	}

	return orm
}

func (s SqlDatabase) Migrate(object interface{}) error {
	// do not migrate invoice, if lightning is not enabled
	if object != nil {
		err := s.db.AutoMigrate(object)
		if err != nil {
			panic(err)
		}
	}
	return nil
}
func (s SqlDatabase) StoreUsedProofs(proof cashu.ProofsUsed) error {
	return s.db.Create(proof).Error
}
func (s SqlDatabase) DeleteProof(proof cashu.Proof) error {

	return s.db.Delete(proof).Error
}
func (s SqlDatabase) ProofsUsed(in []string) []cashu.Proof {
	proofs := make([]cashu.Proof, 0)
	s.db.Where(in).Find(&proofs)
	return proofs
}

func (s SqlDatabase) GetReservedProofs() ([]cashu.Proof, error) {
	proofs := make([]cashu.Proof, 0)
	var tx = s.db.Where("reserved = ?", true)
	tx = tx.Find(&proofs)
	return proofs, tx.Error
}

// GetUsedProofs reads all proofs from db
func (s SqlDatabase) GetUsedProofs() ([]cashu.Proof, error) {
	proofs := make([]cashu.Proof, 0)
	tx := s.db.Find(&proofs)
	return proofs, tx.Error
}

// InvalidateProof will write proof to db
func (s SqlDatabase) StoreProof(p cashu.Proof) error {
	log.WithFields(p.Log()).Info("invalidating proof")
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "secret"}}, // key colume
		DoUpdates: clause.AssignmentColumns([]string{"reserved", "send_id"}),
	}).Create(&p).Error
}

func (s SqlDatabase) GetScripts(address string) ([]cashu.P2SHScript, error) {
	scripts := make([]cashu.P2SHScript, 0)
	var tx = s.db
	if address != "" {
		tx = tx.Where("address = ?", address)
	}
	tx = tx.Find(&scripts)
	return scripts, tx.Error
}
func (s SqlDatabase) StoreScript(p cashu.P2SHScript) error {
	log.Info("storing script")
	return s.db.Create(&p).Error
}

// StorePromise will write promise to db
func (s SqlDatabase) StorePromise(p cashu.Promise) error {
	log.WithFields(p.Log()).Info("storing promise")
	return s.db.Create(&p).Error
}

// StoreLightningInvoice will store lightning invoice in db
func (s SqlDatabase) StoreLightningInvoice(i lightning.Invoice) error {
	log.WithFields(i.Log()).Info("storing lightning invoice")
	return s.db.Create(i).Error
}

// GetLightningInvoices
func (s SqlDatabase) GetLightningInvoices(paid bool) ([]lnbits.Invoice, error) {
	invoices := make([]lnbits.Invoice, 0)
	var tx = s.db.Where("paid = ?", paid)
	tx = tx.Find(&invoices)
	return invoices, tx.Error
}

// GetLightningInvoice reads lighting invoice from db
func (s SqlDatabase) GetLightningInvoice(hash string) (lightning.Invoice, error) {
	invoice := cashu.CreateInvoice()
	invoice.SetHash(hash)
	tx := s.db.Find(invoice)
	log.WithFields(cashuLog.WithLoggable(invoice, tx.Error)).Info("storing lightning invoice")
	return invoice, tx.Error
}

// UpdateLightningInvoice updates lightning invoice in db
func (s SqlDatabase) UpdateLightningInvoice(hash string, options ...UpdateInvoiceOptions) error {
	i, err := s.GetLightningInvoice(hash)
	if err != nil {
		return err
	}
	for _, option := range options {
		option(i)
	}
	return s.db.Save(i).Error
}
