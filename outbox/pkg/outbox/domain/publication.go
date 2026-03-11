package domain

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
)

type publication struct {
	PublicationName  string
	DatabaseName     string
	Tables           []string
	RegisterdActions *Actions
}

func (p publication) doOp(ctx context.Context, conn *pgconn.PgConn, query string) error {
	result := conn.Exec(context.Background(), query)
	_, err := result.ReadAll()
	return err
}

func (p publication) GetPubs() string {
	pubs := "'"
	for _, table := range p.Tables {
		pubs += table + ","
	}
	return pubs[:len(pubs)-1] + "'"
}

func (p publication) Name() string {
	return p.PublicationName
}

type Publication interface {
	DoOp(ctx context.Context, connection *pgconn.PgConn) error
	GetPubs() string
	Name() string
	Actions() *Actions
}

type CreatePublication struct {
	publication
}

func (p CreatePublication) Actions() *Actions {
	return p.RegisterdActions
}

func (p CreatePublication) DoOp(ctx context.Context, connection *pgconn.PgConn) error {
	q := "CREATE PUBLICATION " + p.PublicationName + " FOR TABLE "
	for i, table := range p.Tables {
		q += table
		if i != len(p.Tables)-1 {
			q += ","
		}
	}
	q += ";"
	err := p.doOp(ctx, connection, q)
	if err != nil {
		v, ok := err.(*pgconn.PgError)
		if ok && v.Code == "42710" {
			log.Info().Msgf("publication %s already exists. Ignoring this error as it is expected", p.PublicationName)
			return nil
		}
		log.Error().Err(err).Msgf("create publication error")
	}
	return err
}

type AlterPublication struct {
	publication
}

func (a AlterPublication) DoOp(ctx context.Context, connection *pgconn.PgConn) error {
	q := "ALTER PUBLICATION " + a.PublicationName + " ADD TABLE "
	for i, table := range a.Tables {
		q += table
		if i != len(a.Tables)-1 {
			q += ","
		}
	}
	q += ";"
	err := a.doOp(ctx, connection, q)
	if err != nil {
		// TODO handle an expected error
		log.Error().Err(err).Msgf("alter publication error")
	}
	return err
}
