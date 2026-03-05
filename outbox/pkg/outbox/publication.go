package outbox

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/rs/zerolog/log"
)

type publication struct {
	Name         string
	DatabaseName string
	Tables       []string
}

func (p publication) doOp(ctx context.Context, conn *pgconn.PgConn, query string) error {
	result := conn.Exec(context.Background(), query)
	_, err := result.ReadAll()
	if err != nil {
		log.Error().Err(err).Msgf("create publication error")
	}
	return err
}

type Publication interface {
	DoOp(ctx context.Context, connection *pgconn.PgConn) error
}

type CreatePublication struct {
	publication
}

func (p CreatePublication) DoOp(ctx context.Context, connection *pgconn.PgConn) error {
	q := "CREATE PUBLICATION " + p.Name + " FOR TABLE "
	for i, table := range p.Tables {
		q += table
		if i != len(p.Tables)-1 {
			q += ","
		}
	}
	q += ";"
	err := p.doOp(ctx, connection, q)
	if err != nil {
		log.Error().Err(err).Msgf("create publication error")
	}
	return err
}

type AlterPublication struct {
	publication
}

func (a AlterPublication) DoOp(ctx context.Context, connection *pgconn.PgConn) error {
	q := "ALTER PUBLICATION " + a.Name + " ADD TABLE "
	for i, table := range a.Tables {
		q += table
		if i != len(a.Tables)-1 {
			q += ","
		}
	}
	q += ";"
	err := a.doOp(ctx, connection, q)
	if err != nil {
		log.Error().Err(err).Msgf("alter publication error")
	}
	return err
}
