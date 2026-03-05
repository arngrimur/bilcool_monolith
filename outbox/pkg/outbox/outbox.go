package outbox

import (
	"context"
	"net/url"
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
	W2JoutputPlugin OutputPlugin = "wal2JsonOutput"
)

type Outbox struct {
}

func NewOutbox(ctx context.Context, connStr *url.URL, outputPlugin OutputPlugin, p []Publication) error {
	connStr.Query().Add("replication", "database")
	conn, err := pgconn.Connect(context.Background(), connStr.String())
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	var pluginArguments []string
	if outputPlugin == PgOutputPlugin {
		// streaming of large transactions is available since PG 14 (protocol version 2)
		// we also need to set 'streaming' to 'true'
		pluginArguments = []string{
			"proto_version '2'",
			"publication_names 'bookings_pub'",
			"messages 'true'",
			"streaming 'true'",
		}
	} else if outputPlugin == W2JoutputPlugin {
		pluginArguments = []string{"\"pretty-print\" 'true'"}
	}

	sysident, err := pglogrepl.IdentifySystem(context.Background(), conn)
	if err != nil {
		log.Fatal().Err(err).Msg("IdentifySystem failed:")
	}
	log.Info().Msgf("SystemID: %s, Timeline: %d, XLogPos: %d, DBName: %s", sysident.SystemID, sysident.Timeline, sysident.XLogPos, sysident.DBName)

	slotName := "bookings_pub"

	_, err = pglogrepl.CreateReplicationSlot(context.Background(), conn, slotName, outputPlugin.String(), pglogrepl.CreateReplicationSlotOptions{Temporary: true})
	if err != nil {
		log.Fatal().Err(err).Msgf("CreateReplicationSlot failed")
	}
	log.Info().Msgf("Created temporary replication slot: %s", slotName)

	err = pglogrepl.StartReplication(context.Background(), conn, slotName, sysident.XLogPos, pglogrepl.StartReplicationOptions{PluginArgs: pluginArguments})
	if err != nil {
		log.Fatal().Err(err).Msgf("StartReplication failed")
	}
	log.Info().Msgf("Logical replication started on slot %s", slotName)

	clientXLogPos := sysident.XLogPos
	standbyMessageTimeout := time.Second * 10
	nextStandbyMessageDeadline := time.Now().Add(standbyMessageTimeout)
	relationsV2 := map[uint32]*pglogrepl.RelationMessageV2{}
	typeMap := pgtype.NewMap()

	// whenever we get StreamStartMessage we set inStream to true and then pass it to DecodeV2 function
	// on StreamStopMessage we set it back to false
	inStream := false

	for {
		if time.Now().After(nextStandbyMessageDeadline) {
			err = pglogrepl.SendStandbyStatusUpdate(context.Background(), conn, pglogrepl.StandbyStatusUpdate{WALWritePosition: clientXLogPos})
			if err != nil {
				log.Fatal().Msgf("SendStandbyStatusUpdate failed: %s", err)
			}
			log.Info().Msgf("Sent Standby status message at %s", clientXLogPos)
			nextStandbyMessageDeadline = time.Now().Add(standbyMessageTimeout)
		}

		ctx, cancel := context.WithDeadline(context.Background(), nextStandbyMessageDeadline)
		rawMsg, err := conn.ReceiveMessage(ctx)
		cancel()
		if err != nil {
			if pgconn.Timeout(err) {
				continue
			}
			log.Fatal().Msgf("ReceiveMessage failed: %s", err)
		}

		if errMsg, ok := rawMsg.(*pgproto3.ErrorResponse); ok {
			log.Fatal().Msgf("received Postgres WAL error: %+v", errMsg)
		}

		msg, ok := rawMsg.(*pgproto3.CopyData)
		if !ok {
			log.Info().Msgf("Received unexpected message: %T\n", rawMsg)
			continue
		}

		switch msg.Data[0] {
		case pglogrepl.PrimaryKeepaliveMessageByteID:
			pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(msg.Data[1:])
			if err != nil {
				log.Fatal().Err(err).Msg("ParsePrimaryKeepaliveMessage failed")
			}
			log.Info().Msgf("Primary Keepalive Message => ServerWALEnd: %s ServerTime: %s ReplyRequested: %t", pkm.ServerWALEnd, pkm.ServerTime, pkm.ReplyRequested)
			if pkm.ServerWALEnd > clientXLogPos {
				clientXLogPos = pkm.ServerWALEnd
			}
			if pkm.ReplyRequested {
				nextStandbyMessageDeadline = time.Time{}
			}

		case pglogrepl.XLogDataByteID:
			xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
			if err != nil {
				log.Fatal().Err(err).Msg("ParseXLogData failed")
			}
			if outputPlugin == W2JoutputPlugin {
				log.Info().Msgf("wal2json data: %s\n", string(xld.WALData))
			} else {
				log.Info().Msgf("XLogData => WALStart %s ServerWALEnd %s ServerTime %s WALData:\n", xld.WALStart, xld.ServerWALEnd, xld.ServerTime)
				processV2(xld.WALData, relationsV2, typeMap, &inStream)
			}

			if xld.WALStart > clientXLogPos {
				clientXLogPos = xld.WALStart
			}
		}
	}
}

func processV2(walData []byte, relations map[uint32]*pglogrepl.RelationMessageV2, typeMap *pgtype.Map, inStream *bool) {
	logicalMsg, err := pglogrepl.ParseV2(walData, *inStream)
	if err != nil {
		log.Fatal().Msgf("Parse logical replication message: %s", err)
	}
	log.Info().Msgf("Receive a logical replication message: %s", logicalMsg.Type())
	switch logicalMsg := logicalMsg.(type) {
	case *pglogrepl.RelationMessageV2:
		relations[logicalMsg.RelationID] = logicalMsg

	case *pglogrepl.BeginMessage:
		// Indicates the beginning of a group of changes in a transaction. This is only sent for committed transactions. You won't get any events from rolled back transactions.

	case *pglogrepl.CommitMessage:

	case *pglogrepl.InsertMessageV2:
		rel, ok := relations[logicalMsg.RelationID]
		if !ok {
			log.Fatal().Msgf("unknown relation ID %d", logicalMsg.RelationID)
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
					log.Fatal().Msgf("error decoding column data: %s", err)
				}
				values[colName] = val
			}
		}
		log.Info().Msgf("insert for xid %d\n", logicalMsg.Xid)
		log.Info().Msgf("INSERT INTO %s.%s: %v", rel.Namespace, rel.RelationName, values)

	case *pglogrepl.UpdateMessageV2:
		log.Info().Msgf("update for xid %d\n", logicalMsg.Xid)
		// ...
	case *pglogrepl.DeleteMessageV2:
		log.Info().Msgf("delete for xid %d\n", logicalMsg.Xid)
		// ...
	case *pglogrepl.TruncateMessageV2:
		log.Info().Msgf("truncate for xid %d\n", logicalMsg.Xid)
		// ...

	case *pglogrepl.TypeMessageV2:
	case *pglogrepl.OriginMessage:

	case *pglogrepl.LogicalDecodingMessageV2:
		log.Info().Msgf("Logical decoding message: %q, %q, %d", logicalMsg.Prefix, logicalMsg.Content, logicalMsg.Xid)

	case *pglogrepl.StreamStartMessageV2:
		*inStream = true
		log.Info().Msgf("Stream start message: xid %d, first segment? %d", logicalMsg.Xid, logicalMsg.FirstSegment)
	case *pglogrepl.StreamStopMessageV2:
		*inStream = false
		log.Info().Msgf("Stream stop message")
	case *pglogrepl.StreamCommitMessageV2:
		log.Info().Msgf("Stream commit message: xid %d", logicalMsg.Xid)
	case *pglogrepl.StreamAbortMessageV2:
		log.Info().Msgf("Stream abort message: xid %d", logicalMsg.Xid)
	default:
		log.Info().Msgf("Unknown message type in pgoutput stream: %T", logicalMsg)
	}
}

func decodeTextColumnData(mi *pgtype.Map, data []byte, dataType uint32) (interface{}, error) {
	if dt, ok := mi.TypeForOID(dataType); ok {
		return dt.Codec.DecodeValue(mi, dataType, pgtype.TextFormatCode, data)
	}
	return string(data), nil
}
