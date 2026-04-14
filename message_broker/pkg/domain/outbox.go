package domain

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type OutputPlugin string

func (p OutputPlugin) String() string {
	return string(p)
}

const (
	PgOutputPlugin  OutputPlugin = "pgoutput"
	W2JoutputPlugin OutputPlugin = "wal2json"
)

type Outbox struct {
	mu                         sync.Mutex
	clientXLogPos              pglogrepl.LSN
	standbyMessageTimeout      time.Duration
	nextStandbyMessageDeadline time.Time
	relationsV2                map[uint32]*pglogrepl.RelationMessageV2
	typeMap                    *pgtype.Map
	inStream                   bool
	outputPlugin               OutputPlugin
	conn                       *pgconn.PgConn
	ActionMap                  *Actions
	Publication                Publication
}

func NewOutbox(ctx context.Context, connStr *url.URL, outputPlugin OutputPlugin, p Publication) (*Outbox, error) {
	outboxInstance, err := newOutbox(ctx, connStr, outputPlugin, p)
	if err != nil {
		return nil, err
	}
	outboxInstance.ActionMap = p.Actions()
	return outboxInstance, nil
}

func newOutbox(ctx context.Context, connStr *url.URL, outputPlugin OutputPlugin, p Publication) (*Outbox, error) {
	q := connStr.Query()
	q.Add("replication", "database")
	connStr.RawQuery = q.Encode()

	conn, err := pgconn.Connect(context.Background(), connStr.String())
	if err != nil {
		return nil, err
	}

	err = p.DoOp(ctx, conn)
	if err != nil {
		return nil, err
	}

	sysident, err := pglogrepl.IdentifySystem(context.Background(), conn)
	if err != nil {
		log.Error().Err(err).Msg("IdentifySystem failed:")
	}
	log.Debug().Msgf("SystemID: %s, Timeline: %d, XLogPos: %d, DBName: %s", sysident.SystemID, sysident.Timeline, sysident.XLogPos, sysident.DBName)

	_, err = pglogrepl.CreateReplicationSlot(context.Background(), conn, p.Name(), outputPlugin.String(),
		pglogrepl.CreateReplicationSlotOptions{Temporary: false, Mode: pglogrepl.LogicalReplication})
	if err != nil {
		v, ok := err.(*pgconn.PgError)
		if ok && v.Code == "42710" {
			log.Debug().Msgf("slot %s already exists. Ignoring this error as it is expected", p.Name())
			err = nil
		} else {
			log.Error().Err(err).Msgf("CreateReplicationSlot failed")
			return nil, err
		}
	}
	log.Debug().Msgf("Created temporary replication slot: %s", p.Name())

	standbyMessageTimeout := time.Second * 10
	return &Outbox{
		Publication:                p,
		clientXLogPos:              sysident.XLogPos,
		standbyMessageTimeout:      standbyMessageTimeout,
		nextStandbyMessageDeadline: time.Now().Add(standbyMessageTimeout),
		relationsV2:                map[uint32]*pglogrepl.RelationMessageV2{},
		typeMap:                    pgtype.NewMap(),

		// whenever we get StreamStartMessage we set inStream to true and then pass it to DecodeV2 function
		// on StreamStopMessage we set it back to false
		inStream:     false,
		outputPlugin: outputPlugin,
		conn:         conn,
	}, err

}

func (o *Outbox) StartReplication(ctx context.Context) (chan struct{}, error) {
	stopCh := make(chan struct{})

	var pluginArguments []string
	if o.outputPlugin == PgOutputPlugin {
		// streaming of large transactions is available since PG 14 (protocol version 2)
		// we also need to set 'streaming' to 'true'
		pluginArguments = []string{
			"proto_version '2'",
			fmt.Sprintf("publication_names '%s'", o.Publication.Name()),
			"messages 'true'",
			"streaming 'true'",
		}
	} else if o.outputPlugin == W2JoutputPlugin {
		pluginArguments = []string{"\"pretty-print\" 'true'"}
	}

	err := pglogrepl.StartReplication(context.Background(), o.conn, o.Publication.Name(), o.clientXLogPos, pglogrepl.StartReplicationOptions{PluginArgs: pluginArguments})
	if err != nil {
		log.Error().Err(err).Msgf("StartReplication failed")
		return stopCh, err
	}
	log.Debug().Msgf("Logical replication started on slot %s", o.Publication.Name())
	go func() {
		defer o.conn.Close(context.Background())
		for {
			select {
			case <-stopCh:
				return
			default:
			}
			o.mu.Lock()
			pastDeadline := time.Now().After(o.nextStandbyMessageDeadline)
			walPos := o.clientXLogPos
			o.mu.Unlock()

			if pastDeadline {
				err := pglogrepl.SendStandbyStatusUpdate(context.Background(), o.conn, pglogrepl.StandbyStatusUpdate{WALWritePosition: walPos})
				if err != nil {
					log.Error().Msgf("SendStandbyStatusUpdate failed: %s", err)
				}
				log.Debug().Msgf("Sent Standby status message at %s", walPos)
				o.mu.Lock()
				o.nextStandbyMessageDeadline = time.Now().Add(o.standbyMessageTimeout)
				o.mu.Unlock()
			}

			o.mu.Lock()
			deadline := o.nextStandbyMessageDeadline
			o.mu.Unlock()

			receiveCtx, cancel := context.WithDeadline(context.Background(), deadline)
			rawMsg, err := o.conn.ReceiveMessage(receiveCtx)
			cancel()
			if err != nil {
				if pgconn.Timeout(err) {
					continue
				}
				log.Error().Msgf("ReceiveMessage failed: %s", err)
			}

			if errMsg, ok := rawMsg.(*pgproto3.ErrorResponse); ok {
				log.Error().Msgf("received Postgres WAL error: %+v", errMsg)
			}

			msg, ok := rawMsg.(*pgproto3.CopyData)
			if !ok {
				log.Debug().Msgf("Received unexpected message: %T\n", rawMsg)
				continue
			}

			switch msg.Data[0] {
			case pglogrepl.PrimaryKeepaliveMessageByteID:
				pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(msg.Data[1:])
				if err != nil {
					log.Error().Err(err).Msg("ParsePrimaryKeepaliveMessage failed")
					return
				}
				log.Debug().Msgf("Primary Keepalive Message => ServerWALEnd: %s ServerTime: %s ReplyRequested: %t", pkm.ServerWALEnd, pkm.ServerTime, pkm.ReplyRequested)
				o.mu.Lock()
				if pkm.ServerWALEnd > o.clientXLogPos {
					o.clientXLogPos = pkm.ServerWALEnd
				}
				if pkm.ReplyRequested {
					o.nextStandbyMessageDeadline = time.Time{}
				}
				o.mu.Unlock()

			case pglogrepl.XLogDataByteID:
				log.Debug().Msgf("RECEIVED MESSAGE: %v", msg.Data[0])
				xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
				if err != nil {
					log.Error().Err(err).Msg("ParseXLogData failed")
				}
				if o.outputPlugin == W2JoutputPlugin {
					log.Debug().Msgf("wal2json data: %s\n", string(xld.WALData))
				} else {
					log.Debug().Msgf("XLogData => WALStart %s ServerWALEnd %s ServerTime %s WALData:\n", xld.WALStart, xld.ServerWALEnd, xld.ServerTime)
					o.processV2(ctx, xld.WALData, o.relationsV2, o.typeMap, &o.inStream)
				}

				o.mu.Lock()
				if xld.WALStart > o.clientXLogPos {
					o.clientXLogPos = xld.WALStart
				}
				o.mu.Unlock()
			case pglogrepl.StandbyStatusUpdateByteID:
				log.Debug().Msgf("Received StandbyStatusUpdate message")
			}
		}
	}()
	return stopCh, nil
}

func (o *Outbox) processV2(ctx context.Context, walData []byte, relations map[uint32]*pglogrepl.RelationMessageV2, typeMap *pgtype.Map, inStream *bool) {
	logicalMsg, err := pglogrepl.ParseV2(walData, *inStream)
	if err != nil {
		log.Error().Msgf("Parse logical replication message: %s", err)
		return
	}
	log.Debug().Msgf("Receive a logical replication message: %s", logicalMsg.Type())
	switch logicalMsg := logicalMsg.(type) {
	case *pglogrepl.RelationMessageV2:
		relations[logicalMsg.RelationID] = logicalMsg
	case *pglogrepl.BeginMessage:
		break
		// Indicates the beginning of a group of changes in a transaction. This is only sent for committed transactions. You won't get any events from rolled back transactions.
	case *pglogrepl.CommitMessage:
		for _, rel := range relations {
			t := Table{
				SchemaName: rel.Namespace,
				TableName:  rel.RelationName,
			}
			executeActions(ctx, o.ActionMap.actions[ActionCommit], t)
		}
		break
	case *pglogrepl.InsertMessageV2:
		rel, ok := relations[logicalMsg.RelationID]
		if !ok {
			log.Error().Msgf("unknown relation ID %d", logicalMsg.RelationID)
		}
		values := map[string]interface{}{}
		for idx, col := range logicalMsg.Tuple.Columns {
			colName := rel.Columns[idx].Name
			switch col.DataType {
			case 'n': // null
				values[colName] = nil
			case 'u': // unchanged toast
				// This TOAST value was not changed. TOAST values are not stored in the tuple, and logical replication doesn't want to spend a disk read to fetch its value for you.
			case 't': //text
				val, err := decodeTextColumnData(typeMap, col.Data, rel.Columns[idx].DataType)
				if err != nil {
					log.Error().Msgf("error decoding column data: %s", err)
				}
				values[colName] = val
			}
		}
		log.Debug().Msgf("insert for xid %d\n", logicalMsg.Xid)
		log.Debug().Msgf("INSERT INTO %s.%s: %v", rel.Namespace, rel.RelationName, values)
		executeActions(ctx, o.ActionMap.actions[ActionInsert], Table{
			SchemaName: rel.Namespace,
			TableName:  rel.RelationName,
		})
	case *pglogrepl.UpdateMessageV2:
		log.Debug().Msgf("update for xid %d\n", logicalMsg.Xid)
		break
	case *pglogrepl.DeleteMessageV2:
		log.Debug().Msgf("delete for xid %d\n", logicalMsg.Xid)
		break
	case *pglogrepl.TruncateMessageV2:
		log.Debug().Msgf("truncate for xid %d\n", logicalMsg.Xid)
		break
		// ...

	case *pglogrepl.TypeMessageV2:
		break
	case *pglogrepl.OriginMessage:
		break
	case *pglogrepl.LogicalDecodingMessageV2:
		log.Debug().Msgf("Logical decoding message: %q, %q, %d", logicalMsg.Prefix, logicalMsg.Content, logicalMsg.Xid)
		break
	case *pglogrepl.StreamStartMessageV2:
		*inStream = true
		log.Debug().Msgf("Stream start message: xid %d, first segment? %d", logicalMsg.Xid, logicalMsg.FirstSegment)
		break
	case *pglogrepl.StreamStopMessageV2:
		*inStream = false
		log.Debug().Msgf("Stream stop message")
		break
	case *pglogrepl.StreamCommitMessageV2:
		log.Debug().Msgf("Stream commit message: xid %d", logicalMsg.Xid)
		break
	case *pglogrepl.StreamAbortMessageV2:
		log.Debug().Msgf("Stream abort message: xid %d", logicalMsg.Xid)
		break
	default:
		log.Debug().Msgf("Unknown message type in pgoutput stream: %T", logicalMsg)
	}
}

func executeActions(ctx context.Context, actions []Action, table Table) {
	for _, action := range actions {
		err := action.Execute(ctx, table)
		if err != nil {
			log.Error().Err(err).Str("table", table.SchemaName+"."+table.TableName).Msgf("failed to execute action")
		}
	}
}

func decodeTextColumnData(mi *pgtype.Map, data []byte, dataType uint32) (interface{}, error) {
	if dt, ok := mi.TypeForOID(dataType); ok {
		return dt.Codec.DecodeValue(mi, dataType, pgtype.TextFormatCode, data)
	}
	return string(data), nil
}
