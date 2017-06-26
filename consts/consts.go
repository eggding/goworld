package consts

import "time"

// Tunable Options
const (
	// For Dispatcher
	DISPATCHER_CLIENT_PROXY_WRITE_BUFFER_SIZE = 1024 * 1024 * 2
	DISPATCHER_CLIENT_PROXY_READ_BUFFER_SIZE  = 1024 * 1024 * 2
	DISPATCHER_CLIENT_PROXY_BUFFERED_DELAY    = 10 * time.Millisecond

	// For Server & Gate
	GAME_SERVICE_PACKET_QUEUE_SIZE = 10000 // packet queue size
	// For Server
	SERVER_TICK_INTERVAL = time.Millisecond * 10 // server tick interval => affects timer resolution

	DISPATCHER_CLIENT_WRITE_BUFFER_SIZE = 1024 * 1024 * 2
	DISPATCHER_CLIENT_READ_BUFFER_SIZE  = 1024 * 1024 * 2
	DISPATCHER_CLIENT_BUFFERED_DELAY    = 10 * time.Millisecond

	//SAVE_INTERVAL      = time.Minute * 5 // Save interval of entities

	ENTER_SPACE_REQUEST_TIMEOUT = DISPATCHER_MIGRATE_TIMEOUT + time.Second*5 // enter space should finish in limited seconds
	DISPATCHER_MIGRATE_TIMEOUT  = time.Second * 60
	// For Storage
)

// Debug Options
const (
	DEBUG_PACKETS      = false
	DEBUG_SPACES       = false
	DEBUG_SAVE_LOAD    = false
	DEBUG_CLIENTS      = false
	DEBUG_MIGRATE      = false
	DEBUG_PACKET_ALLOC = false
)