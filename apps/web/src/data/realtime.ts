/**
 * Realtime (WebSocket) client interface — STUB ONLY (M5).
 *
 * The Realtime Gateway implements the wire protocol
 * (WEBSOCKET_PROTOCOL.md) on the server side, but no web client
 * connects to it yet: connection setup, token delivery, subscribe
 * handling, reconnection with last_seq resume, and incoming-event
 * dispatch are all unwired. This module exists so the shape of the
 * future client is declared in one place; it must never pretend a
 * live connection exists.
 *
 * Until the client lands, views keep using REST (listMessagesAsync
 * with since_seq) to catch up. Nothing in the app may import this
 * stub as if it delivered messages.
 */

/** A decoded realtime envelope (WEBSOCKET_PROTOCOL.md §5.1, draft). */
export interface RealtimeEnvelope {
  eventId: string;
  event: string;
  version: string;
  timestamp: string;
  correlationId: string;
  sequence?: number | undefined;
  senderId?: string | undefined;
  data: unknown;
}

export type RealtimeEventHandler = (envelope: RealtimeEnvelope) => void;

/**
 * A realtime connection. The only implementation available today is
 * the disconnected stub returned by createDisconnectedRealtimeClient:
 * every operation rejects or throws with a plain-language reason,
 * and `isConnected` is always false.
 */
export interface RealtimeClient {
  readonly isConnected: boolean;
  connect(): Promise<void>;
  disconnect(): void;
  subscribe(conversationId: string): Promise<void>;
  unsubscribe(conversationId: string): Promise<void>;
  sendTyping(conversationId: string, typing: boolean): Promise<void>;
  onEvent(handler: RealtimeEventHandler): () => void;
}

const NOT_CONNECTED_MESSAGE = '即時連線尚未接線，請以 REST 續拉訊息。';

/** The disconnected stub: honest about never connecting. */
export function createDisconnectedRealtimeClient(): RealtimeClient {
  return {
    isConnected: false,
    connect: async (): Promise<void> => {
      throw new Error(NOT_CONNECTED_MESSAGE);
    },
    disconnect: (): void => undefined,
    subscribe: async (): Promise<void> => {
      throw new Error(NOT_CONNECTED_MESSAGE);
    },
    unsubscribe: async (): Promise<void> => {
      throw new Error(NOT_CONNECTED_MESSAGE);
    },
    sendTyping: async (): Promise<void> => {
      throw new Error(NOT_CONNECTED_MESSAGE);
    },
    onEvent: (): (() => void) => {
      // No events can ever arrive while disconnected; the returned
      // unsubscribe is a no-op.
      return () => undefined;
    },
  };
}
