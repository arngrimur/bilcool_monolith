package domain

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
)

type PublicationBase struct {
	PublicationName   string
	DatabaseName      string
	Tables            []string
	RegisteredActions *Actions
}

func (p PublicationBase) doOp(ctx context.Context, conn *pgconn.PgConn, query string) error {
	result := conn.Exec(context.Background(), query)
	_, err := result.ReadAll()
	return err
}

func (p PublicationBase) GetPubs() string {
	pubs := "'"
	for _, table := range p.Tables {
		pubs += table + ","
	}
	return pubs[:len(pubs)-1] + "'"
}

func (p PublicationBase) Name() string {
	return p.PublicationName
}

type Publication interface {
	DoOp(ctx context.Context, connection *pgconn.PgConn) error
	GetPubs() string
	Name() string
	Actions() *Actions
}

type CreatePublication struct {
	PublicationBase
}

func NewCreatePublications(publicationName, databaseName string, tables []string, actions map[ActionName]Action) CreatePublication {
	c := CreatePublication{
		PublicationBase: PublicationBase{
			PublicationName:   publicationName,
			DatabaseName:      databaseName,
			Tables:            tables,
			RegisteredActions: NewActions(),
		},
	}
	for n, a := range actions {
		c.RegisteredActions.RegisterAction(n, a)
	}
	return c
}

func (p CreatePublication) Actions() *Actions {
	return p.RegisteredActions
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
			log.Info().Msgf("PublicationBase %s already exists. Ignoring this error as it is expected", p.PublicationName)
			return nil
		}
		log.Error().Err(err).Msgf("create PublicationBase error")
	}
	return err
}

type AlterPublication struct {
	PublicationBase
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
		log.Error().Err(err).Msgf("alter PublicationBase error")
	}
	return err
}
