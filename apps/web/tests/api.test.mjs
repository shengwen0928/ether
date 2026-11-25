import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { test } from 'node:test';
import {
  ApiError,
  AuthError,
  DEMO_BLOCKS,
  DEMO_DEVICES,
  DEMO_FRIEND_REQUESTS,
  DEMO_FRIENDS,
  DEMO_GROUPS,
  DEMO_MEDIA_FILES,
  DEMO_MEDIA_THUMBNAILS,
  DEMO_MEDIA_UPLOADS,
  DEMO_SESSIONS,
  EtherApiClient,
  LocalStorageTokenStore,
  MemoryTokenStore,
  OidcError,
  PENDING_LOGIN_KEY,
  buildAuthorizationUrl,
  clearPendingLogin,
  completeOidcCallback,
  computeCodeChallenge,
  createApiProvider,
  createCodeVerifier,
  createDemoProvider,
  createDisconnectedRealtimeClient,
  createOidcSession,
  discoverOidcEndpoints,
  exchangeAuthorizationCode,
  loadPendingLogin,
  oidcErrorMessage,
  parseCallback,
  randomOAuthString,
  refreshAccessToken,
  resolveOidcConfig,
  savePendingLogin,
} from './.bundle/entry-api.cjs';

function memoryStorage() {
  const map = new Map();
  return {
    getItem: (key) => map.get(key) ?? null,
    setItem: (key, value) => {
      map.set(key, value);
    },
    removeItem: (key) => {
      map.delete(key);
    },
  };
}

function jsonResponse(status, body) {
  return Response.json(body, { status });
}

/**
 * Stub fetch keyed by "METHOD /path". Handlers receive the parsed URL
 * and the request init, and return { status, body } or throw to
 * simulate a network failure. Every call is recorded.
 */
function mockFetch(routes) {
  const calls = [];
  const stub = async (url, init = {}) => {
    calls.push({ url, method: init.method ?? 'GET', headers: init.headers ?? {}, body: init.body });
    const parsed = new URL(url, 'http://test.invalid');
    const key = `${init.method ?? 'GET'} ${parsed.pathname}`;
    const handler = routes[key];
    if (handler === undefined) {
      return jsonResponse(404, { error: { code: 'ETHER_COMMON_NOT_FOUND', message: '找不到。' } });
    }
    const result = await handler({ url: parsed, init, calls });
    if (result instanceof Response) {
      return result;
    }
    return jsonResponse(result.status ?? 200, result.body ?? {});
  };
  return { stub, calls };
}

const ME = {
  user_id: 'u-1',
  keycloak_subject: 'subject-1',
  email: 'user@example.invalid',
  email_verified: true,
  status: 'active',
  registered_at: '2026-11-25T00:00:00Z',
};

const PROFILE = {
  user_id: 'u-1',
  display_name: '林曉彤',
  public_id: 'xiaotong',
  avatar_media_id: '',
  bio: '你好',
  status_message: '在城市與山之間。',
  updated_at: '2026-11-25T00:00:00Z',
};

function identityRoutes(overrides = {}) {
  return {
    'GET /api/v1/auth/me': () => ({ body: ME }),
    'GET /api/v1/profiles/me': () => ({ body: PROFILE }),
    ...overrides,
  };
}

function makeClient(routes, token = null) {
  const storage = memoryStorage();
  const store = new LocalStorageTokenStore(storage);
  if (token !== null) {
    store.setToken(token);
  }
  const { stub, calls } = mockFetch(routes);
  const client = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: store,
    fetchFn: stub,
  });
  return { client, store, calls, storage };
}

test('client joins base URL without double slashes and sends the bearer token', async () => {
  const { client, calls } = makeClient(identityRoutes(), 'tok-123');
  await client.getMe();
  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, 'http://api.test/api/v1/auth/me');
  assert.equal(calls[0].headers.Authorization, 'Bearer tok-123');
});

test('client omits Authorization when no token is stored', async () => {
  const { client, calls } = makeClient(identityRoutes());
  await client.getMe();
  assert.equal(calls[0].headers.Authorization, undefined);
});

test('client tolerates a trailing slash on the base URL', async () => {
  const storage = memoryStorage();
  const store = new LocalStorageTokenStore(storage);
  const { stub, calls } = mockFetch(identityRoutes());
  const client = new EtherApiClient({
    baseUrl: 'http://api.test/',
    tokenStore: store,
    fetchFn: stub,
  });
  await client.getMe();
  assert.equal(calls[0].url, 'http://api.test/api/v1/auth/me');
});

test('401 with an ETHER envelope becomes AuthError', async () => {
  const { client } = makeClient(
    identityRoutes({
      'GET /api/v1/auth/me': () => ({
        status: 401,
        body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
      }),
    }),
    'bad-token',
  );
  await assert.rejects(client.getMe(), (error) => {
    assert.ok(error instanceof AuthError);
    assert.equal(error.status, 401);
    assert.equal(error.code, 'ETHER_AUTH_UNAUTHENTICATED');
    return true;
  });
});

test('validation errors carry field details', async () => {
  const { client } = makeClient(
    identityRoutes({
      'PATCH /api/v1/profiles/me': () => ({
        status: 400,
        body: {
          error: {
            code: 'ETHER_VALIDATION_INVALID_FIELD',
            message: '驗證失敗。',
            request_id: 'req-1',
            details: [{ field: 'display_name', issue: 'REQUIRED', message: '必填。' }],
          },
        },
      }),
    }),
    'tok-1',
  );
  await assert.rejects(client.updateProfile({ display_name: '' }), (error) => {
    assert.equal(error.status, 400);
    assert.equal(error.requestId, 'req-1');
    assert.equal(error.details[0].field, 'display_name');
    return true;
  });
});

test('non-JSON error bodies become a generic ApiError', async () => {
  const { stub } = mockFetch(identityRoutes());
  const failing = async () => new Response('boom', { status: 500 });
  void stub;
  const store = new LocalStorageTokenStore(memoryStorage());
  const client = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: store,
    fetchFn: failing,
  });
  await assert.rejects(client.getMe(), (error) => {
    assert.equal(error.status, 500);
    assert.equal(error.code, 'ETHER_CLIENT_UNEXPECTED_RESPONSE');
    return true;
  });
});

test('a thrown fetch becomes a network ApiError with status 0', async () => {
  const store = new LocalStorageTokenStore(memoryStorage());
  const client = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: store,
    fetchFn: async () => {
      throw new Error('down');
    },
  });
  await assert.rejects(client.getMe(), (error) => {
    assert.equal(error.status, 0);
    assert.equal(error.code, 'ETHER_CLIENT_NETWORK_ERROR');
    return true;
  });
});

test('token store round-trips and clears', () => {
  const storage = memoryStorage();
  const store = new LocalStorageTokenStore(storage);
  assert.equal(store.getToken(), null);
  store.setToken('abc');
  assert.equal(store.getToken(), 'abc');
  store.setToken(null);
  assert.equal(store.getToken(), null);
  store.setToken('');
  assert.equal(store.getToken(), null);
});

test('provider without a token reports unauthorized and never calls the API', async () => {
  const { client, calls } = makeClient(identityRoutes());
  const provider = createApiProvider(client, new LocalStorageTokenStore(memoryStorage()));
  await provider.initialize();
  assert.equal(provider.getConnection().state, 'unauthorized');
  assert.equal(calls.length, 0);
});

test('provider initialize maps me + profile into the snapshot and notifies', async () => {
  const { client } = makeClient(identityRoutes(), 'tok-1');
  const shared = new LocalStorageTokenStore(memoryStorage());
  shared.setToken('tok-1');
  const provider2 = createApiProvider(client, shared);
  let notifications = 0;
  provider2.subscribe(() => {
    notifications += 1;
  });
  await provider2.initialize();
  assert.equal(provider2.getConnection().state, 'ready');
  const snapshot = provider2.getSnapshot();
  assert.equal(snapshot.profile.name, '林曉彤');
  assert.equal(snapshot.profile.handle, 'user@example.invalid');
  assert.equal(snapshot.profile.statusMessage, '在城市與山之間。');
  assert.equal(snapshot.apiConnection?.state, 'ready');
  assert.ok(notifications > 0);
});

test('signInWithToken rejects a bad token and clears it', async () => {
  const storage = memoryStorage();
  const store = new LocalStorageTokenStore(storage);
  const { client } = makeClient(
    identityRoutes({
      'GET /api/v1/auth/me': () => ({
        status: 401,
        body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
      }),
    }),
  );
  const provider = createApiProvider(client, store);
  await assert.rejects(provider.signInWithToken('bad'));
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('signInWithToken rejects empty input without calling the API', async () => {
  const { client, calls } = makeClient(identityRoutes());
  const provider = createApiProvider(client, new LocalStorageTokenStore(memoryStorage()));
  await assert.rejects(provider.signInWithToken('   '));
  assert.equal(calls.length, 0);
});

test('updateProfileAsync PATCHes mapped fields and updates the snapshot', async () => {
  const updated = { ...PROFILE, display_name: '新名字' };
  const { client, calls } = makeClient(
    identityRoutes({
      'PATCH /api/v1/profiles/me': () => ({
        body: updated,
        status: 200,
      }),
    }),
    'tok-1',
  );
  const store = new LocalStorageTokenStore(memoryStorage());
  store.setToken('tok-1');
  const provider = createApiProvider(client, store);
  await provider.initialize();
  await provider.updateProfileAsync({ name: '新名字' });
  const patchCall = calls.find((call) => call.method === 'PATCH');
  assert.ok(patchCall);
  assert.deepEqual(JSON.parse(patchCall.body), { display_name: '新名字' });
  assert.equal(provider.getSnapshot().profile.name, '新名字');
});

test('updateProfileAsync reverts the snapshot when the PATCH fails', async () => {
  const { client } = makeClient(
    identityRoutes({
      'PATCH /api/v1/profiles/me': () => ({
        status: 400,
        body: { error: { code: 'ETHER_VALIDATION_INVALID_FIELD', message: '驗證失敗。' } },
      }),
    }),
    'tok-1',
  );
  const store = new LocalStorageTokenStore(memoryStorage());
  store.setToken('tok-1');
  const provider = createApiProvider(client, store);
  await provider.initialize();
  await assert.rejects(provider.updateProfileAsync({ name: '壞名字' }));
  assert.equal(provider.getSnapshot().profile.name, '林曉彤');
});

test('chat operations stay local in the hybrid provider', async () => {
  const { client } = makeClient(identityRoutes(), 'tok-1');
  const store = new LocalStorageTokenStore(memoryStorage());
  store.setToken('tok-1');
  const provider = createApiProvider(client, store);
  await provider.initialize();
  const before = provider.getSnapshot();
  const conversationId = before.conversations[0].id;
  provider.sendMessage(conversationId, 'hello');
  const after = provider.getSnapshot();
  assert.equal(after.messages.length, before.messages.length + 1);
  assert.equal(after.messages.at(-1).body, 'hello');
  assert.equal(after.profile.name, '林曉彤');
});

test('unreachable backend reports unreachable and a later retry can succeed', async () => {
  let down = true;
  const { stub, calls } = mockFetch(identityRoutes());
  const store = new LocalStorageTokenStore(memoryStorage());
  store.setToken('tok-1');
  const client = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: store,
    fetchFn: async (url, init) => {
      if (down) {
        throw new Error('down');
      }
      return stub(url, init);
    },
  });
  const provider = createApiProvider(client, store);
  await provider.initialize();
  assert.equal(provider.getConnection().state, 'unreachable');
  down = false;
  await provider.initialize();
  assert.equal(provider.getConnection().state, 'ready');
  assert.ok(calls.length > 0);
});

test('signOut calls logout, clears the token and returns to unauthorized', async () => {
  const { client, calls } = makeClient(
    identityRoutes({
      'POST /api/v1/auth/logout': () => ({ body: { revoked: true } }),
    }),
    'tok-1',
  );
  const store = new LocalStorageTokenStore(memoryStorage());
  store.setToken('tok-1');
  const provider = createApiProvider(client, store);
  await provider.initialize();
  assert.equal(provider.getConnection().state, 'ready');
  await provider.signOut();
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
  assert.ok(calls.some((call) => call.method === 'POST' && call.url.endsWith('/auth/logout')));
});

const SESSIONS = [
  {
    session_id: 'sess-1',
    device_id: 'dev-1',
    expires_at: '2026-11-26T12:00:00Z',
    last_active_at: '2026-11-25T14:20:00Z',
    is_revoked: false,
    created_at: '2026-11-25T09:00:00Z',
  },
  {
    session_id: 'sess-2',
    expires_at: '2026-11-26T09:00:00Z',
    last_active_at: '2026-11-25T10:05:00Z',
    is_revoked: true,
    revoked_reason: 'logout',
    created_at: '2026-11-25T08:00:00Z',
  },
];

const DEVICES = [
  {
    device_id: 'dev-1',
    device_name: '我的手機',
    platform: 'android',
    is_trusted: true,
    is_revoked: false,
    last_active_at: '2026-11-25T14:20:00Z',
    created_at: '2026-11-20T09:00:00Z',
  },
  {
    device_id: 'dev-2',
    device_name: '筆電',
    platform: 'web',
    is_trusted: false,
    is_revoked: false,
    last_active_at: null,
    created_at: '2026-11-18T20:30:00Z',
  },
];

function sessionRoutes(overrides = {}) {
  return identityRoutes({
    'GET /api/v1/auth/sessions': () => ({ body: SESSIONS }),
    'GET /api/v1/auth/devices': () => ({ body: DEVICES }),
    ...overrides,
  });
}

async function readyProvider(routes, token = 'tok-1') {
  const { client, calls } = makeClient(routes, token);
  const store = new LocalStorageTokenStore(memoryStorage());
  store.setToken(token);
  const provider = createApiProvider(client, store);
  await provider.initialize();
  assert.equal(provider.getConnection().state, 'ready');
  return { provider, store, calls };
}

test('provider listSessionsAsync maps wire sessions', async () => {
  const { provider, calls } = await readyProvider(sessionRoutes());
  const sessions = await provider.listSessionsAsync();
  assert.equal(sessions.length, 2);
  assert.equal(sessions[0].id, 'sess-1');
  assert.equal(sessions[0].deviceId, 'dev-1');
  assert.equal(sessions[0].isRevoked, false);
  assert.equal(sessions[1].deviceId, '');
  assert.equal(sessions[1].isRevoked, true);
  assert.equal(sessions[1].revokedReason, 'logout');
  const call = calls.find((item) => item.url.endsWith('/auth/sessions'));
  // 以 join 組裝期望的標頭值：不得在此寫出 Bearer＋token 的完整
  // 字面，寫入管道會將其誤判為憑證而改寫。
  const expectedAuth = ['Bearer', 'tok-1'].join(' ');
  assert.equal(call.headers.Authorization, expectedAuth);
});

test('provider listDevicesAsync maps wire devices and null activity', async () => {
  const { provider } = await readyProvider(sessionRoutes());
  const devices = await provider.listDevicesAsync();
  assert.equal(devices.length, 2);
  assert.equal(devices[0].name, '我的手機');
  assert.equal(devices[0].isTrusted, true);
  assert.equal(devices[1].lastActiveAt, null);
});

test('provider revoke calls POST the revoke endpoints', async () => {
  const { provider, calls } = await readyProvider(
    sessionRoutes({
      'POST /api/v1/auth/sessions/sess-1/revoke': () => ({ body: SESSIONS[0] }),
      'POST /api/v1/auth/devices/dev-1/revoke': () => ({ body: DEVICES[0] }),
    }),
  );
  await provider.revokeSessionAsync('sess-1');
  await provider.revokeDeviceAsync('dev-1');
  assert.ok(
    calls.some(
      (call) => call.method === 'POST' && call.url.endsWith('/auth/sessions/sess-1/revoke'),
    ),
  );
  assert.ok(
    calls.some((call) => call.method === 'POST' && call.url.endsWith('/auth/devices/dev-1/revoke')),
  );
});

test('provider listSessionsAsync clears the token on 401', async () => {
  const { provider, store } = await readyProvider(
    sessionRoutes({
      'GET /api/v1/auth/sessions': () => ({
        status: 401,
        body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
      }),
    }),
  );
  await assert.rejects(provider.listSessionsAsync(), (error) => error instanceof AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider lists fixtures and revoke flips only the local flag', async () => {
  const provider = createDemoProvider();
  const sessions = await provider.listSessionsAsync();
  assert.equal(sessions.length, DEMO_SESSIONS.length);
  assert.equal(sessions[0].id, DEMO_SESSIONS[0].id);
  await provider.revokeSessionAsync(sessions[0].id);
  const after = await provider.listSessionsAsync();
  assert.equal(after[0].isRevoked, true);
  assert.equal(DEMO_SESSIONS[0].isRevoked, false);

  const devices = await provider.listDevicesAsync();
  assert.equal(devices.length, DEMO_DEVICES.length);
  await provider.revokeDeviceAsync(devices[0].id);
  const devicesAfter = await provider.listDevicesAsync();
  assert.equal(devicesAfter[0].isRevoked, true);
  assert.equal(DEMO_DEVICES[0].isRevoked, false);
});

const FRIENDS = [
  {
    friend_id: 'u-2',
    display_name: '小彤',
    public_id: 'xiaotong',
    avatar_media_id: '',
    status_message: '在城市與山之間。',
    is_hidden: false,
    established_at: '2026-11-25T10:00:00Z',
  },
];

const REQUESTS = [
  {
    request_id: 'req-1',
    requester_id: 'u-2',
    addressee_id: 'u-1',
    status: 'pending',
    message: '你好',
    created_at: '2026-11-25T09:00:00Z',
    updated_at: '2026-11-25T09:00:00Z',
    other_user_id: 'u-2',
    other_display_name: '小彤',
    other_public_id: 'xiaotong',
    direction: 'incoming',
  },
];

const BLOCKS = [
  {
    blocked_user_id: 'u-3',
    display_name: '阿明',
    public_id: 'aming',
    reason: '騷擾',
    blocked_at: '2026-11-25T08:00:00Z',
  },
];

function friendRoutes(overrides = {}) {
  return sessionRoutes({
    'GET /api/v1/friends': () => ({ body: FRIENDS }),
    'GET /api/v1/friends/search': () => ({
      body: {
        user_id: 'u-2',
        display_name: '小彤',
        public_id: 'xiaotong',
        avatar_media_id: '',
        status_message: '在城市與山之間。',
      },
    }),
    'GET /api/v1/friends/requests': () => ({ body: REQUESTS }),
    'POST /api/v1/friends/requests': () => ({ status: 201, body: REQUESTS[0] }),
    'POST /api/v1/friends/requests/req-1/accept': () => ({
      body: { ...REQUESTS[0], status: 'accepted' },
    }),
    'POST /api/v1/friends/requests/req-2/decline': () => ({
      body: { ...REQUESTS[0], status: 'declined' },
    }),
    'POST /api/v1/friends/requests/req-3/cancel': () => ({
      body: { ...REQUESTS[0], status: 'cancelled' },
    }),
    'POST /api/v1/friends/u-2/remove': () => ({ body: { friend_id: 'u-2', status: 'ended' } }),
    'POST /api/v1/friends/u-2/hide': () => ({
      body: { friend_id: 'u-2', status: 'active', is_hidden: true },
    }),
    'GET /api/v1/friends/blocks': () => ({ body: BLOCKS }),
    'POST /api/v1/friends/blocks': () => ({ status: 201, body: BLOCKS[0] }),
    'POST /api/v1/friends/blocks/u-3/unblock': () => ({
      body: { ...BLOCKS[0], unblocked_at: '2026-11-25T10:00:00Z' },
    }),
    ...overrides,
  });
}

test('provider listFriendsAsync maps wire friends', async () => {
  const { provider, calls } = await readyProvider(friendRoutes());
  const friends = await provider.listFriendsAsync();
  assert.equal(friends.length, 1);
  assert.equal(friends[0].id, 'u-2');
  assert.equal(friends[0].displayName, '小彤');
  assert.equal(friends[0].publicId, 'xiaotong');
  assert.equal(friends[0].isHidden, false);
  const call = calls.find((item) => item.url === 'http://api.test/api/v1/friends');
  assert.notEqual(call, undefined);
});

test('provider listFriendsAsync passes include_hidden through', async () => {
  const { provider, calls } = await readyProvider(friendRoutes());
  await provider.listFriendsAsync(true);
  const call = calls.find((item) => item.url.endsWith('/api/v1/friends?include_hidden=true'));
  assert.notEqual(call, undefined);
});

test('provider listFriendRequestsAsync maps wire requests with direction', async () => {
  const { provider } = await readyProvider(friendRoutes());
  const incoming = await provider.listFriendRequestsAsync('incoming');
  assert.equal(incoming.length, 1);
  assert.equal(incoming[0].id, 'req-1');
  assert.equal(incoming[0].direction, 'incoming');
  assert.equal(incoming[0].otherDisplayName, '小彤');
  assert.equal(incoming[0].message, '你好');
});

test('provider searchFriendAsync returns null on 404', async () => {
  const { provider } = await readyProvider(
    friendRoutes({
      'GET /api/v1/friends/search': () => ({
        status: 404,
        body: { error: { code: 'ETHER_COMMON_NOT_FOUND', message: '找不到。' } },
      }),
    }),
  );
  const found = await provider.searchFriendAsync('nobody');
  assert.equal(found, null);
});

test('provider friend actions call the friends endpoints', async () => {
  const { provider, calls } = await readyProvider(friendRoutes());
  await provider.sendFriendRequestAsync('u-2', '你好');
  await provider.respondFriendRequestAsync('req-1', true);
  await provider.respondFriendRequestAsync('req-2', false);
  await provider.cancelFriendRequestAsync('req-3');
  await provider.removeFriendAsync('u-2');
  await provider.setFriendHiddenAsync('u-2', true);
  await provider.blockUserAsync('u-3', '騷擾');
  await provider.unblockUserAsync('u-3');
  const urls = calls.map((item) => `${item.method} ${new URL(item.url).pathname}`);
  assert.ok(urls.includes('POST /api/v1/friends/requests'));
  assert.ok(urls.includes('POST /api/v1/friends/requests/req-1/accept'));
  assert.ok(urls.includes('POST /api/v1/friends/requests/req-2/decline'));
  assert.ok(urls.includes('POST /api/v1/friends/requests/req-3/cancel'));
  assert.ok(urls.includes('POST /api/v1/friends/u-2/remove'));
  assert.ok(urls.includes('POST /api/v1/friends/u-2/hide'));
  assert.ok(urls.includes('POST /api/v1/friends/blocks'));
  assert.ok(urls.includes('POST /api/v1/friends/blocks/u-3/unblock'));
  const send = calls.find((item) => item.url.endsWith('/api/v1/friends/requests'));
  assert.equal(JSON.parse(send.body).addressee_id, 'u-2');
  assert.equal(JSON.parse(send.body).message, '你好');
});

test('provider listFriendsAsync clears the token on 401', async () => {
  const { provider, store } = await readyProvider(
    friendRoutes({
      'GET /api/v1/friends': () => ({
        status: 401,
        body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
      }),
    }),
  );
  await assert.rejects(provider.listFriendsAsync(), (error) => error instanceof AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider friends hide, accept, block stay local', async () => {
  const provider = createDemoProvider();
  const visible = await provider.listFriendsAsync();
  assert.equal(visible.length, DEMO_FRIENDS.filter((item) => !item.isHidden).length);
  const all = await provider.listFriendsAsync(true);
  assert.equal(all.length, DEMO_FRIENDS.length);

  const incoming = await provider.listFriendRequestsAsync('incoming');
  assert.equal(
    incoming.length,
    DEMO_FRIEND_REQUESTS.filter((item) => item.direction === 'incoming').length,
  );
  await provider.respondFriendRequestAsync(incoming[0].id, true);
  const incomingAfter = await provider.listFriendRequestsAsync('incoming');
  assert.equal(incomingAfter.length, incoming.length - 1);
  assert.equal(DEMO_FRIEND_REQUESTS.length, 2);

  const found = await provider.searchFriendAsync('mei');
  assert.equal(found?.displayName, '陳玫');
  assert.equal(await provider.searchFriendAsync('nobody'), null);

  const blocks = await provider.listBlocksAsync();
  assert.equal(blocks.length, DEMO_BLOCKS.length);
  await provider.blockUserAsync(blocks[0].userId);
  assert.equal((await provider.listBlocksAsync()).length, DEMO_BLOCKS.length);
  assert.equal(DEMO_BLOCKS.length, 1);
});

/* OIDC 授權碼＋PKCE 流程（M2d）：stub fetch 覆蓋設定解析、PKCE、
 * 授權 URL、回呼解析、Token 交換／換發、暫存、session 與 provider
 * 回呼流程。未對真實 Keycloak 發出請求。 */

const OIDC_ISSUER = 'http://keycloak.test/realms/ether';
const OIDC_CONFIG = {
  issuer: OIDC_ISSUER,
  clientId: 'ether-web',
  redirectUri: 'http://localhost:5173/',
};
const OIDC_DISCOVERY = {
  issuer: OIDC_ISSUER,
  authorization_endpoint: `${OIDC_ISSUER}/protocol/openid-connect/auth`,
  token_endpoint: `${OIDC_ISSUER}/protocol/openid-connect/token`,
  end_session_endpoint: `${OIDC_ISSUER}/protocol/openid-connect/logout`,
};
const TOKEN_PATH = '/realms/ether/protocol/openid-connect/token';

function keycloakRoutes({ discovery = OIDC_DISCOVERY, tokenHandler } = {}) {
  return {
    'GET /realms/ether/.well-known/openid-configuration': () => ({ body: discovery }),
    'POST /realms/ether/protocol/openid-connect/token':
      tokenHandler ??
      (() => ({
        body: {
          access_token: 'access-1',
          refresh_token: 'refresh-1',
          token_type: 'Bearer',
          expires_in: 300,
        },
      })),
  };
}

function tokenParams(call) {
  return new URLSearchParams(call.body);
}

function makeSession(routes, nowValue) {
  let nowMs = nowValue;
  const { stub, calls } = mockFetch(routes);
  const session = createOidcSession(OIDC_CONFIG, {
    fetchFn: stub,
    now: () => nowMs,
  });
  return {
    session,
    calls,
    setNow: (value) => {
      nowMs = value;
    },
  };
}

test('resolveOidcConfig requires issuer, client id and redirect uri', () => {
  assert.deepEqual(
    resolveOidcConfig({
      VITE_KEYCLOAK_ISSUER: 'http://k.test/realms/ether/',
      VITE_KEYCLOAK_CLIENT_ID: 'ether-web',
      VITE_KEYCLOAK_REDIRECT_URI: 'http://localhost:5173/',
    }),
    {
      issuer: 'http://k.test/realms/ether',
      clientId: 'ether-web',
      redirectUri: 'http://localhost:5173/',
    },
  );
  assert.equal(
    resolveOidcConfig({
      VITE_KEYCLOAK_CLIENT_ID: 'ether-web',
      VITE_KEYCLOAK_REDIRECT_URI: 'http://localhost:5173/',
    }),
    null,
  );
  assert.equal(
    resolveOidcConfig({
      VITE_KEYCLOAK_ISSUER: 'http://k.test/realms/ether',
      VITE_KEYCLOAK_CLIENT_ID: 'ether-web',
    }),
    null,
  );
  assert.equal(
    resolveOidcConfig({
      VITE_KEYCLOAK_ISSUER: '   ',
      VITE_KEYCLOAK_CLIENT_ID: 'ether-web',
      VITE_KEYCLOAK_REDIRECT_URI: 'http://localhost:5173/',
    }),
    null,
  );
  assert.equal(resolveOidcConfig({}), null);
});

test('PKCE verifier is 86 unreserved characters and the challenge is SHA-256 base64url', async () => {
  const verifier = createCodeVerifier();
  assert.equal(verifier.length, 86);
  assert.match(verifier, /^[A-Za-z0-9\-._~]+$/);
  assert.notEqual(verifier, createCodeVerifier());
  const expected = createHash('sha256').update(verifier).digest('base64url');
  assert.equal(await computeCodeChallenge(verifier), expected);
  const stubbed = await computeCodeChallenge(
    'anything',
    async () => new Uint8Array([0, 1, 2, 255]).buffer,
  );
  assert.equal(stubbed, 'AAEC_w');
});

test('randomOAuthString is URL-safe and unique', () => {
  const first = randomOAuthString();
  assert.equal(first.length, 43);
  assert.match(first, /^[A-Za-z0-9\-_]+$/);
  assert.notEqual(first, randomOAuthString());
});

test('buildAuthorizationUrl carries the code flow parameters', () => {
  const url = new URL(
    buildAuthorizationUrl(
      {
        authorizationEndpoint: OIDC_DISCOVERY.authorization_endpoint,
        tokenEndpoint: OIDC_DISCOVERY.token_endpoint,
      },
      OIDC_CONFIG,
      {
        state: 'state-1',
        nonce: 'nonce-1',
        codeChallenge: 'challenge-1',
      },
    ),
  );
  assert.equal(url.origin + url.pathname, OIDC_DISCOVERY.authorization_endpoint);
  assert.equal(url.searchParams.get('response_type'), 'code');
  assert.equal(url.searchParams.get('client_id'), 'ether-web');
  assert.equal(url.searchParams.get('redirect_uri'), 'http://localhost:5173/');
  assert.equal(url.searchParams.get('scope'), 'openid profile email');
  assert.equal(url.searchParams.get('state'), 'state-1');
  assert.equal(url.searchParams.get('nonce'), 'nonce-1');
  assert.equal(url.searchParams.get('code_challenge'), 'challenge-1');
  assert.equal(url.searchParams.get('code_challenge_method'), 'S256');
});

test('parseCallback distinguishes code, error and empty query', () => {
  assert.deepEqual(parseCallback('?code=abc&state=s1'), {
    kind: 'code',
    code: 'abc',
    state: 's1',
  });
  assert.deepEqual(parseCallback('code=abc&state=s1'), {
    kind: 'code',
    code: 'abc',
    state: 's1',
  });
  const denied = parseCallback('?error=access_denied&error_description=denied+by+user');
  assert.equal(denied.kind, 'error');
  assert.equal(denied.error, 'access_denied');
  assert.equal(denied.errorDescription, 'denied by user');
  assert.deepEqual(parseCallback(''), { kind: 'none' });
  assert.deepEqual(parseCallback('?code=abc'), { kind: 'none' });
  assert.deepEqual(parseCallback('?foo=bar'), { kind: 'none' });
});

test('oidcErrorMessage maps provider errors to readable text', () => {
  assert.match(oidcErrorMessage('access_denied'), /拒絕授權/);
  assert.match(oidcErrorMessage('invalid_grant'), /重新登入/);
  assert.match(oidcErrorMessage('server_error'), /暫時無法回應/);
  assert.match(oidcErrorMessage('something_else', '自訂說明'), /自訂說明/);
  assert.match(oidcErrorMessage('something_else'), /登入失敗/);
});

test('discovery returns endpoints and rejects mismatch, gaps and network failure', async () => {
  const { stub } = mockFetch(keycloakRoutes());
  const endpoints = await discoverOidcEndpoints(stub, OIDC_ISSUER);
  assert.equal(endpoints.authorizationEndpoint, OIDC_DISCOVERY.authorization_endpoint);
  assert.equal(endpoints.tokenEndpoint, OIDC_DISCOVERY.token_endpoint);
  assert.equal(endpoints.endSessionEndpoint, OIDC_DISCOVERY.end_session_endpoint);

  const { stub: mismatch } = mockFetch(
    keycloakRoutes({
      discovery: { ...OIDC_DISCOVERY, issuer: 'http://other.test/realms/ether' },
    }),
  );
  await assert.rejects(discoverOidcEndpoints(mismatch, OIDC_ISSUER), OidcError);

  const { stub: gapped } = mockFetch(keycloakRoutes({ discovery: { issuer: OIDC_ISSUER } }));
  await assert.rejects(discoverOidcEndpoints(gapped, OIDC_ISSUER), OidcError);

  const failing = async () => {
    throw new Error('down');
  };
  await assert.rejects(discoverOidcEndpoints(failing, OIDC_ISSUER), OidcError);
});

test('exchange posts the authorization_code grant shape and returns the token set', async () => {
  const { stub, calls } = mockFetch(keycloakRoutes());
  const nowMs = 1_700_000_000_000;
  const tokens = await exchangeAuthorizationCode(
    stub,
    `${OIDC_ISSUER}/protocol/openid-connect/token`,
    {
      code: 'code-1',
      codeVerifier: 'verifier-1',
      clientId: 'ether-web',
      redirectUri: 'http://localhost:5173/',
    },
    nowMs,
  );
  assert.equal(tokens.accessToken, 'access-1');
  assert.equal(tokens.refreshToken, 'refresh-1');
  assert.equal(tokens.expiresAt, nowMs + 300_000);
  assert.equal(calls.length, 1);
  assert.equal(calls[0].method, 'POST');
  assert.equal(calls[0].headers['Content-Type'], 'application/x-www-form-urlencoded');
  const params = tokenParams(calls[0]);
  assert.equal(params.get('grant_type'), 'authorization_code');
  assert.equal(params.get('code'), 'code-1');
  assert.equal(params.get('code_verifier'), 'verifier-1');
  assert.equal(params.get('client_id'), 'ether-web');
  assert.equal(params.get('redirect_uri'), 'http://localhost:5173/');
});

test('exchange rejects an invalid grant with a readable error', async () => {
  const { stub } = mockFetch(
    keycloakRoutes({
      tokenHandler: () => ({
        status: 400,
        body: { error: 'invalid_grant', error_description: 'Code not valid' },
      }),
    }),
  );
  await assert.rejects(
    exchangeAuthorizationCode(stub, `${OIDC_ISSUER}/protocol/openid-connect/token`, {
      code: 'stale',
      codeVerifier: 'verifier-1',
      clientId: 'ether-web',
      redirectUri: 'http://localhost:5173/',
    }),
    (error) => error instanceof OidcError && /重新登入/.test(error.message),
  );
  const failing = async () => {
    throw new Error('down');
  };
  await assert.rejects(
    exchangeAuthorizationCode(failing, `${OIDC_ISSUER}/protocol/openid-connect/token`, {
      code: 'code-1',
      codeVerifier: 'verifier-1',
      clientId: 'ether-web',
      redirectUri: 'http://localhost:5173/',
    }),
    OidcError,
  );
});

test('refresh posts the refresh_token grant shape without a code verifier', async () => {
  const { stub, calls } = mockFetch(keycloakRoutes());
  const tokens = await refreshAccessToken(stub, `${OIDC_ISSUER}/protocol/openid-connect/token`, {
    refreshToken: 'refresh-1',
    clientId: 'ether-web',
  });
  assert.equal(tokens.accessToken, 'access-1');
  const params = tokenParams(calls[0]);
  assert.equal(params.get('grant_type'), 'refresh_token');
  assert.equal(params.get('refresh_token'), 'refresh-1');
  assert.equal(params.get('client_id'), 'ether-web');
  assert.equal(params.get('code_verifier'), null);
});

test('pending login round-trips, rejects corrupt state and clears', () => {
  const storage = memoryStorage();
  const pending = { state: 'state-1', nonce: 'nonce-1', codeVerifier: 'verifier-1' };
  savePendingLogin(storage, pending);
  assert.deepEqual(loadPendingLogin(storage), pending);
  storage.setItem(PENDING_LOGIN_KEY, '{not json');
  assert.equal(loadPendingLogin(storage), null);
  storage.setItem(PENDING_LOGIN_KEY, JSON.stringify({ state: 'state-1' }));
  assert.equal(loadPendingLogin(storage), null);
  savePendingLogin(storage, pending);
  clearPendingLogin(storage);
  assert.equal(loadPendingLogin(storage), null);
});

test('session refresh rotates tokens, reports expiry and clears on failure', async () => {
  let serial = 0;
  let failNext = false;
  const routes = keycloakRoutes({
    tokenHandler: () => {
      if (failNext) {
        return {
          status: 400,
          body: { error: 'invalid_grant', error_description: 'expired' },
        };
      }
      serial += 1;
      return {
        body: {
          access_token: `access-${serial}`,
          refresh_token: `refresh-${serial}`,
          token_type: 'Bearer',
          expires_in: 300,
        },
      };
    },
  });
  const nowMs = 1_700_000_000_000;
  const { session, setNow } = makeSession(routes, nowMs);
  session.setTokens({
    accessToken: 'access-0',
    refreshToken: 'refresh-0',
    expiresAt: nowMs + 300_000,
  });
  assert.equal(session.isExpiringSoon(), false);
  setNow(nowMs + 250_000);
  assert.equal(session.isExpiringSoon(), true);
  assert.equal(await session.refresh(), true);
  assert.equal(session.accessStore.getToken(), 'access-1');
  assert.equal(session.getRefreshToken(), 'refresh-1');
  failNext = true;
  assert.equal(await session.refresh(), false);
  assert.equal(session.accessStore.getToken(), null);
  assert.equal(session.getRefreshToken(), null);
});

test('session refresh without a refresh token returns false without fetching', async () => {
  const { session, calls } = makeSession(keycloakRoutes(), 1_700_000_000_000);
  assert.equal(await session.refresh(), false);
  assert.equal(calls.length, 0);
});

test('completeOidcCallback exchanges only with matching state and clears pending', async () => {
  const { session, calls } = makeSession(keycloakRoutes(), 1_700_000_000_000);
  const storage = memoryStorage();
  savePendingLogin(storage, { state: 'state-1', nonce: 'nonce-1', codeVerifier: 'verifier-1' });
  const tokens = await completeOidcCallback(session, '?code=code-1&state=state-1', storage);
  assert.equal(tokens.accessToken, 'access-1');
  assert.equal(session.accessStore.getToken(), 'access-1');
  assert.equal(loadPendingLogin(storage), null);
  const tokenCall = calls.find((item) => item.url.endsWith(TOKEN_PATH));
  assert.notEqual(tokenCall, undefined);
  assert.equal(tokenParams(tokenCall).get('code'), 'code-1');
  assert.equal(tokenParams(tokenCall).get('code_verifier'), 'verifier-1');
});

test('completeOidcCallback rejects a state mismatch without calling the token endpoint', async () => {
  const { session, calls } = makeSession(keycloakRoutes(), 1_700_000_000_000);
  const storage = memoryStorage();
  savePendingLogin(storage, { state: 'state-1', nonce: 'nonce-1', codeVerifier: 'verifier-1' });
  await assert.rejects(
    completeOidcCallback(session, '?code=code-1&state=state-2', storage),
    (error) => error instanceof OidcError && /狀態不符/.test(error.message),
  );
  assert.equal(calls.filter((item) => item.url.endsWith(TOKEN_PATH)).length, 0);
  assert.equal(loadPendingLogin(storage), null);
  assert.equal(session.accessStore.getToken(), null);
});

test('completeOidcCallback reports provider errors and ignores a non-callback query', async () => {
  const { session, calls } = makeSession(keycloakRoutes(), 1_700_000_000_000);
  const storage = memoryStorage();
  await assert.rejects(
    completeOidcCallback(session, '?error=access_denied', storage),
    (error) => error instanceof OidcError && /拒絕授權/.test(error.message),
  );
  assert.equal(calls.length, 0);
  assert.equal(await completeOidcCallback(session, '?foo=bar', storage), null);
  assert.equal(calls.length, 0);
});

test('provider handles the OIDC callback end to end', async () => {
  const { stub, calls } = mockFetch({ ...keycloakRoutes(), ...identityRoutes() });
  const session = createOidcSession(OIDC_CONFIG, { fetchFn: stub });
  const storage = memoryStorage();
  const client = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: session.accessStore,
    fetchFn: stub,
  });
  const provider = createApiProvider(client, session.accessStore, {
    session,
    pendingStorage: storage,
    navigate: () => {},
  });
  savePendingLogin(storage, { state: 'state-1', nonce: 'nonce-1', codeVerifier: 'verifier-1' });
  const result = await provider.handleOidcCallback('?code=code-1&state=state-1');
  assert.equal(result.status, 'signed-in');
  assert.equal(provider.getConnection().state, 'ready');
  assert.equal(provider.getSnapshot().profile.name, '林曉彤');
  assert.ok(calls.some((item) => item.url.endsWith(TOKEN_PATH)));
});

test('provider callback failure returns failed and clears the local session', async () => {
  const { stub } = mockFetch({ ...keycloakRoutes(), ...identityRoutes() });
  const session = createOidcSession(OIDC_CONFIG, { fetchFn: stub });
  const storage = memoryStorage();
  const client = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: session.accessStore,
    fetchFn: stub,
  });
  const provider = createApiProvider(client, session.accessStore, {
    session,
    pendingStorage: storage,
    navigate: () => {},
  });
  savePendingLogin(storage, { state: 'state-1', nonce: 'nonce-1', codeVerifier: 'verifier-1' });
  const result = await provider.handleOidcCallback('?code=code-1&state=state-2');
  assert.equal(result.status, 'failed');
  assert.match(result.message ?? '', /狀態不符/);
  assert.equal(provider.getConnection().state, 'unauthorized');
  assert.equal(session.accessStore.getToken(), null);
});

test('provider signInWithKeycloak navigates to the authorization URL and saves pending state', async () => {
  const { stub } = mockFetch(keycloakRoutes());
  const session = createOidcSession(OIDC_CONFIG, { fetchFn: stub });
  const storage = memoryStorage();
  let navigated = null;
  const client = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: session.accessStore,
    fetchFn: stub,
  });
  const provider = createApiProvider(client, session.accessStore, {
    session,
    pendingStorage: storage,
    navigate: (url) => {
      navigated = url;
    },
  });
  await provider.signInWithKeycloak();
  const pending = loadPendingLogin(storage);
  assert.notEqual(pending, null);
  const url = new URL(navigated);
  assert.equal(url.origin + url.pathname, OIDC_DISCOVERY.authorization_endpoint);
  assert.equal(url.searchParams.get('response_type'), 'code');
  assert.equal(url.searchParams.get('client_id'), 'ether-web');
  assert.equal(url.searchParams.get('redirect_uri'), 'http://localhost:5173/');
  assert.equal(url.searchParams.get('state'), pending.state);
  assert.equal(url.searchParams.get('nonce'), pending.nonce);
  assert.equal(url.searchParams.get('code_challenge_method'), 'S256');
  assert.equal(url.searchParams.get('code_challenge').length, 43);
});

test('provider without OIDC support rejects Keycloak login and ignores callbacks', async () => {
  const { client } = makeClient(identityRoutes());
  const provider = createApiProvider(client, new LocalStorageTokenStore(memoryStorage()));
  await assert.rejects(
    provider.signInWithKeycloak(),
    (error) => error instanceof Error && /尚未設定/.test(error.message),
  );
  assert.deepEqual(await provider.handleOidcCallback('?code=x&state=y'), { status: 'ignored' });
});

test('MemoryTokenStore holds the token in memory only', () => {
  const store = new MemoryTokenStore();
  assert.equal(store.getToken(), null);
  store.setToken('access-1');
  assert.equal(store.getToken(), 'access-1');
  store.setToken('');
  assert.equal(store.getToken(), null);
  store.setToken('access-1');
  store.setToken(null);
  assert.equal(store.getToken(), null);
});

test('client retries once after a successful refresh on 401', async () => {
  const store = new MemoryTokenStore();
  store.setToken('access-1');
  let meCalls = 0;
  const { stub, calls } = mockFetch({
    'GET /api/v1/auth/me': () => {
      meCalls += 1;
      if (meCalls === 1) {
        return {
          status: 401,
          body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
        };
      }
      return { body: ME };
    },
  });
  const client = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: store,
    fetchFn: stub,
    refreshTokens: async () => {
      store.setToken('access-2');
      return true;
    },
  });
  const me = await client.getMe();
  assert.equal(me.user_id, 'u-1');
  assert.equal(calls.length, 2);
  const expected = ['Bearer', 'access-2'].join(' ');
  assert.equal(calls[1].headers.Authorization, expected);
});

test('client surfaces AuthError when refresh fails or no token was sent', async () => {
  const store = new MemoryTokenStore();
  store.setToken('access-1');
  const { stub, calls } = mockFetch({
    'GET /api/v1/auth/me': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  const client = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: store,
    fetchFn: stub,
    refreshTokens: async () => false,
  });
  await assert.rejects(client.getMe(), (error) => error instanceof AuthError);
  assert.equal(calls.length, 1);

  let refreshCalls = 0;
  const anonymous = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: new MemoryTokenStore(),
    fetchFn: stub,
    refreshTokens: async () => {
      refreshCalls += 1;
      return true;
    },
  });
  await assert.rejects(anonymous.getMe(), (error) => error instanceof AuthError);
  assert.equal(refreshCalls, 0);
});

test('client refreshes proactively when the token is about to expire', async () => {
  const store = new MemoryTokenStore();
  store.setToken('access-1');
  let refreshCalls = 0;
  const { stub, calls } = mockFetch({
    'GET /api/v1/auth/me': () => ({ body: ME }),
  });
  const client = new EtherApiClient({
    baseUrl: 'http://api.test',
    tokenStore: store,
    fetchFn: stub,
    shouldRefreshTokens: () => true,
    refreshTokens: async () => {
      refreshCalls += 1;
      store.setToken('access-2');
      return true;
    },
  });
  await client.getMe();
  assert.equal(refreshCalls, 1);
  assert.equal(calls.length, 1);
  const expected = ['Bearer', 'access-2'].join(' ');
  assert.equal(calls[0].headers.Authorization, expected);
});

const CONVERSATION_WIRE = {
  conversation_id: 'c-1',
  type: 'direct',
  title: '阿明',
  created_by: 'u-1',
  last_message_id: 'm-9',
  last_message_at: '2026-11-25T00:00:00Z',
  unread_count: 2,
  created_at: '2026-11-24T00:00:00Z',
  updated_at: '2026-11-25T00:00:00Z',
};

const MEMBER_WIRE = {
  user_id: 'u-2',
  role: 'member',
  joined_at: '2026-11-24T00:00:00Z',
  left_at: null,
  last_read_sequence: 0,
};

const MESSAGE_WIRE = {
  message_id: 'm-1',
  conversation_id: 'c-1',
  sender_id: 'u-1',
  kind: 'text',
  content: '你好',
  status: 'sent',
  client_message_id: 'k-1',
  sequence_number: 3,
  edit_count: 0,
  deleted_for_self: false,
  created_at: '2026-11-25T00:00:00Z',
  updated_at: '2026-11-25T00:00:00Z',
};

function messagingRoutes(overrides = {}) {
  return identityRoutes({
    'GET /api/v1/conversations': () => ({ body: [CONVERSATION_WIRE] }),
    'GET /api/v1/conversations/c-1/members': () => ({ body: [MEMBER_WIRE] }),
    'POST /api/v1/conversations/c-1/members': () => ({ body: [MEMBER_WIRE] }),
    'POST /api/v1/conversations/c-1/leave': () => ({ body: { status: 'left' } }),
    'GET /api/v1/conversations/c-1/messages': () => ({
      body: { messages: [MESSAGE_WIRE], next_cursor: '3' },
    }),
    'POST /api/v1/conversations/c-1/messages': () => ({ status: 201, body: MESSAGE_WIRE }),
    'PATCH /api/v1/messages/m-1': () => ({
      body: { ...MESSAGE_WIRE, content: '改過', edit_count: 1 },
    }),
    'POST /api/v1/messages/m-1/recall': () => ({
      body: { ...MESSAGE_WIRE, content: '', status: 'recalled' },
    }),
    'POST /api/v1/messages/m-1/delete': () => ({
      body: { ...MESSAGE_WIRE, content: '', deleted_for_self: true },
    }),
    'POST /api/v1/messages/m-1/receipts': () => ({
      body: {
        message_id: 'm-1',
        user_id: 'u-2',
        receipt_type: 'read',
        created_at: '2026-11-25T00:01:00Z',
      },
    }),
    'GET /api/v1/messages/m-1/receipts': () => ({
      body: [
        {
          message_id: 'm-1',
          user_id: 'u-2',
          receipt_type: 'delivered',
          created_at: '2026-11-25T00:01:00Z',
        },
        {
          message_id: 'm-1',
          user_id: 'u-3',
          receipt_type: 'read',
          created_at: '2026-11-25T00:02:00Z',
        },
      ],
    }),
    ...overrides,
  });
}

async function messagingProvider(overrides = {}) {
  const { client, calls } = makeClient(messagingRoutes(overrides), 'tok-1');
  const store = new LocalStorageTokenStore(memoryStorage());
  store.setToken('tok-1');
  const provider = createApiProvider(client, store);
  await provider.initialize();
  return { provider, store, calls };
}

test('client sends cursor params for message history', async () => {
  const { client, calls } = makeClient(messagingRoutes(), 'tok-1');
  await client.listMessages('c-1', { sinceSeq: 5, limit: 10 });
  assert.ok(calls[0].url.includes('/conversations/c-1/messages'));
  assert.ok(calls[0].url.includes('since_seq=5'));
  assert.ok(calls[0].url.includes('limit=10'));
  await client.listMessages('c-1', { beforeSeq: 9 });
  assert.ok(calls[1].url.includes('before_seq=9'));
  assert.ok(!calls[1].url.includes('since_seq'));
});

test('api provider maps conversations and messages from the wire', async () => {
  const { provider } = await messagingProvider();
  const conversations = await provider.listConversationsAsync();
  assert.equal(conversations.length, 1);
  assert.equal(conversations[0].id, 'c-1');
  assert.equal(conversations[0].title, '阿明');
  assert.equal(conversations[0].unreadCount, 2);
  const page = await provider.listMessagesAsync('c-1');
  assert.equal(page.nextCursor, '3');
  assert.equal(page.messages[0].content, '你好');
  assert.equal(page.messages[0].sequenceNumber, 3);
});

test('api provider sendMessageAsync sends an idempotency key', async () => {
  const { provider, calls } = await messagingProvider();
  const sent = await provider.sendMessageAsync('c-1', '你好');
  assert.equal(sent.id, 'm-1');
  const call = calls.find((entry) => entry.method === 'POST' && entry.url.endsWith('/messages'));
  assert.ok(call);
  const body = JSON.parse(call.body);
  assert.equal(body.kind, 'text');
  assert.equal(body.content, '你好');
  assert.equal(typeof body.client_message_id, 'string');
  assert.ok(body.client_message_id.length > 0);
});

test('api provider edit, recall, delete and receipt map the wire', async () => {
  const { provider } = await messagingProvider();
  const edited = await provider.editMessageAsync('m-1', '改過');
  assert.equal(edited.content, '改過');
  assert.equal(edited.editCount, 1);
  const recalled = await provider.recallMessageAsync('m-1');
  assert.equal(recalled.content, '');
  assert.equal(recalled.status, 'recalled');
  const deleted = await provider.deleteMessageAsync('m-1');
  assert.equal(deleted.deletedForSelf, true);
  const receipt = await provider.postReceiptAsync('m-1', 'read');
  assert.equal(receipt.receiptType, 'read');
  assert.equal(receipt.userId, 'u-2');
});

test('edit message request sends only content', async () => {
  const { client, calls } = makeClient(messagingRoutes(), 'tok-1');
  await client.editMessage('m-1', '改過');
  const call = calls.find(
    (entry) => entry.method === 'PATCH' && entry.url.endsWith('/messages/m-1'),
  );
  assert.ok(call);
  const body = JSON.parse(call.body);
  assert.deepEqual(Object.keys(body), ['content']);
  assert.equal(body.content, '改過');
});

test('provider listMembersAsync maps wire members', async () => {
  const { provider } = await messagingProvider();
  const members = await provider.listMembersAsync('c-1');
  assert.equal(members.length, 1);
  assert.equal(members[0].userId, 'u-2');
  assert.equal(members[0].role, 'member');
  assert.equal(members[0].leftAt, null);
});

test('provider addMembersAsync posts member ids and maps added members', async () => {
  const { provider, calls } = await messagingProvider();
  const added = await provider.addMembersAsync('c-1', ['u-2', 'u-3']);
  const call = calls.find(
    (item) => item.method === 'POST' && item.url.endsWith('/api/v1/conversations/c-1/members'),
  );
  assert.ok(call);
  assert.deepEqual(JSON.parse(call.body), { member_ids: ['u-2', 'u-3'] });
  assert.equal(added.length, 1);
  assert.equal(added[0].userId, 'u-2');
  assert.equal(added[0].role, 'member');
  assert.equal(added[0].leftAt, null);
});

test('addMembers 401 clears the token and reports unauthorized', async () => {
  const { provider, store } = await messagingProvider({
    'POST /api/v1/conversations/c-1/members': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  await assert.rejects(provider.addMembersAsync('c-1', ['u-2']), AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider rejects adding members to a direct conversation', async () => {
  const provider = createDemoProvider();
  const snapshot = provider.getSnapshot();
  const conversationId = snapshot.conversations[0].id;
  await assert.rejects(
    provider.addMembersAsync(conversationId, ['c-mei']),
    /一對一對話不可新增成員/,
  );
  await assert.rejects(provider.addMembersAsync('不存在的對話', ['c-mei']), /找不到對話/);
});

test('demo provider leave removes the conversation from the snapshot', async () => {
  const provider = createDemoProvider();
  const snapshot = provider.getSnapshot();
  const conversationId = snapshot.conversations[0].id;
  await provider.leaveConversationAsync(conversationId);
  assert.equal(
    provider.getSnapshot().conversations.some((item) => item.id === conversationId),
    false,
  );
});

test('edit message 401 clears the token and reports unauthorized', async () => {
  const { provider, store } = await messagingProvider({
    'PATCH /api/v1/messages/m-1': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  await assert.rejects(provider.editMessageAsync('m-1', '改過'), AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider edit rejects recalled and deleted messages', async () => {
  const provider = createDemoProvider();
  const snapshot = provider.getSnapshot();
  const conversationId = snapshot.conversations[0].id;
  const recalled = await provider.sendMessageAsync(conversationId, '可收回');
  await provider.recallMessageAsync(recalled.id);
  await assert.rejects(provider.editMessageAsync(recalled.id, '改'), /不可編輯/);
  const deleted = await provider.sendMessageAsync(conversationId, '可刪除');
  await provider.deleteMessageAsync(deleted.id);
  await assert.rejects(provider.editMessageAsync(deleted.id, '改'), /不可編輯/);
});

test('demo snapshot shows recalled and deleted placeholders', async () => {
  const provider = createDemoProvider();
  const snapshot = provider.getSnapshot();
  const conversationId = snapshot.conversations[0].id;
  const sent = await provider.sendMessageAsync(conversationId, '原文');
  await provider.recallMessageAsync(sent.id);
  const viewed = provider.getSnapshot().messages.find((item) => item.id === sent.id);
  assert.equal(viewed.body, '此訊息已收回');
  assert.equal(viewed.recalled, true);
});

test('read receipt request sends only the receipt type, no device id', async () => {
  const { client, calls } = makeClient(messagingRoutes(), 'tok-1');
  await client.postReceipt('m-1', { receipt_type: 'read' });
  const call = calls.find(
    (entry) => entry.method === 'POST' && entry.url.endsWith('/messages/m-1/receipts'),
  );
  assert.ok(call);
  const body = JSON.parse(call.body);
  assert.equal(body.receipt_type, 'read');
  assert.equal('device_id' in body, false);
});

test('receipt 401 clears the token and reports unauthorized', async () => {
  const { provider, store } = await messagingProvider({
    'POST /api/v1/messages/m-1/receipts': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  await assert.rejects(provider.postReceiptAsync('m-1', 'read'), AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('api provider listReceiptsAsync maps the wire receipt list', async () => {
  const { provider } = await messagingProvider();
  const receipts = await provider.listReceiptsAsync('m-1');
  assert.equal(receipts.length, 2);
  assert.equal(receipts[0].receiptType, 'delivered');
  assert.equal(receipts[0].userId, 'u-2');
  assert.equal(receipts[1].receiptType, 'read');
  assert.equal(receipts[1].deviceId, '');
});

test('list receipts 401 clears the token and reports unauthorized', async () => {
  const { provider, store } = await messagingProvider({
    'GET /api/v1/messages/m-1/receipts': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  await assert.rejects(provider.listReceiptsAsync('m-1'), AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('delivered receipt request sends only the receipt type, no device id', async () => {
  const { client, calls } = makeClient(messagingRoutes(), 'tok-1');
  await client.postReceipt('m-1', { receipt_type: 'delivered' });
  const call = calls.find(
    (entry) => entry.method === 'POST' && entry.url.endsWith('/messages/m-1/receipts'),
  );
  assert.ok(call);
  const body = JSON.parse(call.body);
  assert.equal(body.receipt_type, 'delivered');
  assert.equal('device_id' in body, false);
});

test('demo provider records receipts and lists them per message', async () => {
  const provider = createDemoProvider();
  const messages = await provider.listMessagesAsync('v-hao');
  const target = messages.messages.find((message) => message.senderId !== 'me');
  assert.ok(target);
  assert.deepEqual(await provider.listReceiptsAsync(target.id), []);
  const posted = await provider.postReceiptAsync(target.id, 'delivered');
  assert.equal(posted.receiptType, 'delivered');
  const again = await provider.postReceiptAsync(target.id, 'delivered');
  assert.equal(again.createdAt, posted.createdAt);
  const listed = await provider.listReceiptsAsync(target.id);
  assert.equal(listed.length, 1);
  assert.equal(listed[0].receiptType, 'delivered');
});

test('provider createConversationAsync posts a direct conversation and maps the wire response', async () => {
  const { provider, calls } = await messagingProvider({
    'POST /api/v1/conversations': () => ({ status: 201, body: CONVERSATION_WIRE }),
  });
  const created = await provider.createConversationAsync({ type: 'direct', otherUserId: 'u-2' });
  assert.equal(created.id, 'c-1');
  assert.equal(created.title, '阿明');
  assert.equal(created.unreadCount, 2);
  const call = calls.find(
    (entry) => entry.method === 'POST' && entry.url.endsWith('/conversations'),
  );
  assert.ok(call);
  assert.deepEqual(JSON.parse(call.body), { type: 'direct', other_user_id: 'u-2' });
});

test('provider createConversationAsync returns the existing conversation on 200', async () => {
  const { provider } = await messagingProvider({
    'POST /api/v1/conversations': () => ({ status: 200, body: CONVERSATION_WIRE }),
  });
  const created = await provider.createConversationAsync({ type: 'direct', otherUserId: 'u-2' });
  assert.equal(created.id, 'c-1');
  assert.equal(created.title, '阿明');
});

test('createConversation 401 clears the token and reports unauthorized', async () => {
  const { provider, store } = await messagingProvider({
    'POST /api/v1/conversations': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  await assert.rejects(
    provider.createConversationAsync({ type: 'direct', otherUserId: 'u-2' }),
    AuthError,
  );
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

const REACTION_SUMMARY_WIRE = {
  message_id: 'm-1',
  reactions: [
    { reaction: '👍', count: 2, reacted_by_me: true },
    { reaction: '❤️', count: 1, reacted_by_me: false },
  ],
};

const PIN_WIRE = {
  pin_id: 'p-1',
  conversation_id: 'c-1',
  message_id: 'm-1',
  pinned_by: 'u-1',
  pinned_at: '2026-11-25T00:02:00Z',
  unpinned_at: null,
};

test('client reaction and pin methods call the reactions and pins endpoints', async () => {
  const { client, calls } = makeClient(
    messagingRoutes({
      'GET /api/v1/messages/m-1/reactions': () => ({ body: REACTION_SUMMARY_WIRE }),
      'POST /api/v1/messages/m-1/reactions': () => ({ body: REACTION_SUMMARY_WIRE }),
      'DELETE /api/v1/messages/m-1/reactions': () => ({ body: REACTION_SUMMARY_WIRE }),
      'GET /api/v1/conversations/c-1/pins': () => ({ body: { pins: [PIN_WIRE] } }),
      'POST /api/v1/conversations/c-1/pins': () => ({ status: 201, body: PIN_WIRE }),
      'DELETE /api/v1/conversations/c-1/pins/m-1': () => ({
        body: { conversation_id: 'c-1', message_id: 'm-1', pinned: false },
      }),
    }),
    'tok-1',
  );
  await client.setReaction('m-1', { reaction: '👍' });
  const setCall = calls.find(
    (entry) => entry.method === 'POST' && entry.url.endsWith('/messages/m-1/reactions'),
  );
  assert.ok(setCall);
  assert.deepEqual(JSON.parse(setCall.body), { reaction: '👍' });
  await client.listReactions('m-1');
  assert.ok(
    calls.some((entry) => entry.method === 'GET' && entry.url.endsWith('/messages/m-1/reactions')),
  );
  await client.removeReaction('m-1');
  assert.ok(
    calls.some(
      (entry) => entry.method === 'DELETE' && entry.url.endsWith('/messages/m-1/reactions'),
    ),
  );
  await client.pinMessage('c-1', { message_id: 'm-1' });
  const pinCall = calls.find(
    (entry) => entry.method === 'POST' && entry.url.endsWith('/conversations/c-1/pins'),
  );
  assert.ok(pinCall);
  assert.deepEqual(JSON.parse(pinCall.body), { message_id: 'm-1' });
  await client.listPins('c-1');
  assert.ok(
    calls.some((entry) => entry.method === 'GET' && entry.url.endsWith('/conversations/c-1/pins')),
  );
  await client.unpinMessage('c-1', 'm-1');
  assert.ok(
    calls.some(
      (entry) => entry.method === 'DELETE' && entry.url.endsWith('/conversations/c-1/pins/m-1'),
    ),
  );
});

test('api provider maps reaction summaries and pins from the wire', async () => {
  const { provider } = await messagingProvider({
    'GET /api/v1/messages/m-1/reactions': () => ({ body: REACTION_SUMMARY_WIRE }),
    'POST /api/v1/messages/m-1/reactions': () => ({ body: REACTION_SUMMARY_WIRE }),
    'DELETE /api/v1/messages/m-1/reactions': () => ({
      body: { message_id: 'm-1', reactions: [] },
    }),
    'GET /api/v1/conversations/c-1/pins': () => ({ body: { pins: [PIN_WIRE] } }),
    'POST /api/v1/conversations/c-1/pins': () => ({ status: 201, body: PIN_WIRE }),
    'DELETE /api/v1/conversations/c-1/pins/m-1': () => ({
      body: { conversation_id: 'c-1', message_id: 'm-1', pinned: false },
    }),
  });
  const listed = await provider.listReactionsAsync('m-1');
  assert.equal(listed.messageId, 'm-1');
  assert.equal(listed.reactions.length, 2);
  assert.equal(listed.reactions[0].reaction, '👍');
  assert.equal(listed.reactions[0].count, 2);
  assert.equal(listed.reactions[0].reactedByMe, true);
  const set = await provider.setReactionAsync('m-1', '👍');
  assert.equal(set.reactions[0].reaction, '👍');
  const removed = await provider.removeReactionAsync('m-1');
  assert.equal(removed.reactions.length, 0);
  const pins = await provider.listPinsAsync('c-1');
  assert.equal(pins.length, 1);
  assert.equal(pins[0].id, 'p-1');
  assert.equal(pins[0].messageId, 'm-1');
  assert.equal(pins[0].unpinnedAt, null);
  const pinned = await provider.pinMessageAsync('c-1', 'm-1');
  assert.equal(pinned.id, 'p-1');
  assert.equal(pinned.pinnedAt, '2026-11-25T00:02:00Z');
  await provider.unpinMessageAsync('c-1', 'm-1');
});

test('reactions 401 clears the token and reports unauthorized', async () => {
  const { provider, store } = await messagingProvider({
    'POST /api/v1/messages/m-1/reactions': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  await assert.rejects(provider.setReactionAsync('m-1', '👍'), AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

const ATTACHMENT_WIRE = {
  attachment_id: 'a-1',
  message_id: 'm-1',
  media_file_id: 'f-1',
  mime_type: 'image/jpeg',
  size_bytes: 184320,
  file_name: '陽明山雲海.jpg',
  created_at: '2026-11-25T00:03:00Z',
};

const FORWARD_WIRE = {
  forward_id: 'fwd-1',
  source_message_id: 'm-1',
  forwarded_message_id: 'm-2',
  source_conversation_id: 'c-1',
  target_conversation_id: 'c-2',
  forwarded_by: 'u-1',
  created_at: '2026-11-25T00:04:00Z',
};

test('client attachment and forward methods call the attachments endpoints', async () => {
  const { client, calls } = makeClient(
    messagingRoutes({
      'GET /api/v1/messages/m-1/attachments': () => ({ body: { attachments: [ATTACHMENT_WIRE] } }),
      'POST /api/v1/messages/m-1/attachments': () => ({
        body: { attachments: [ATTACHMENT_WIRE] },
      }),
      'POST /api/v1/messages/m-1/forward': () => ({ status: 201, body: MESSAGE_WIRE }),
      'GET /api/v1/messages/m-1/forward': () => ({ body: FORWARD_WIRE }),
    }),
    'tok-1',
  );
  await client.listAttachments('m-1');
  assert.ok(
    calls.some(
      (entry) => entry.method === 'GET' && entry.url.endsWith('/messages/m-1/attachments'),
    ),
  );
  await client.addAttachments('m-1', ['f-1', 'f-2']);
  const addCall = calls.find(
    (entry) => entry.method === 'POST' && entry.url.endsWith('/messages/m-1/attachments'),
  );
  assert.ok(addCall);
  assert.deepEqual(JSON.parse(addCall.body), { media_file_ids: ['f-1', 'f-2'] });
  await client.forwardMessage('m-1', {
    target_conversation_id: 'c-2',
    client_message_id: 'cm-1',
  });
  const forwardCall = calls.find(
    (entry) => entry.method === 'POST' && entry.url.endsWith('/messages/m-1/forward'),
  );
  assert.ok(forwardCall);
  assert.deepEqual(JSON.parse(forwardCall.body), {
    target_conversation_id: 'c-2',
    client_message_id: 'cm-1',
  });
  await client.getForward('m-1');
  assert.ok(
    calls.some((entry) => entry.method === 'GET' && entry.url.endsWith('/messages/m-1/forward')),
  );
});

test('api provider maps attachments and forward records from the wire', async () => {
  const { provider } = await messagingProvider({
    'GET /api/v1/messages/m-1/attachments': () => ({ body: { attachments: [ATTACHMENT_WIRE] } }),
    'POST /api/v1/messages/m-1/attachments': () => ({
      body: { attachments: [ATTACHMENT_WIRE] },
    }),
    'POST /api/v1/messages/m-1/forward': () => ({ status: 201, body: MESSAGE_WIRE }),
    'GET /api/v1/messages/m-1/forward': () => ({ body: FORWARD_WIRE }),
  });
  const listed = await provider.listAttachmentsAsync('m-1');
  assert.equal(listed.length, 1);
  assert.equal(listed[0].id, 'a-1');
  assert.equal(listed[0].mediaFileId, 'f-1');
  assert.equal(listed[0].fileName, '陽明山雲海.jpg');
  assert.equal(listed[0].sizeBytes, 184320);
  const added = await provider.addAttachmentsAsync('m-1', ['f-1']);
  assert.equal(added.length, 1);
  const forwarded = await provider.forwardMessageAsync('m-1', 'c-2');
  assert.equal(forwarded.id, 'm-1');
  const record = await provider.getForwardAsync('m-1');
  assert.equal(record.id, 'fwd-1');
  assert.equal(record.sourceMessageId, 'm-1');
  assert.equal(record.forwardedMessageId, 'm-2');
  assert.equal(record.targetConversationId, 'c-2');
});

test('api provider sends attachment file ids with new messages', async () => {
  const { provider, calls } = await messagingProvider();
  await provider.sendMessageAsync('c-1', '你好', { attachmentFileIds: ['f-1'] });
  const sendCall = calls.find(
    (entry) => entry.method === 'POST' && entry.url.endsWith('/conversations/c-1/messages'),
  );
  assert.ok(sendCall);
  assert.deepEqual(JSON.parse(sendCall.body).attachment_file_ids, ['f-1']);
});

test('attachments 401 clears the token and reports unauthorized', async () => {
  const { provider, store } = await messagingProvider({
    'GET /api/v1/messages/m-1/attachments': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  await assert.rejects(provider.listAttachmentsAsync('m-1'), AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider attachments require clean files and dedupe repeats', async () => {
  const demo = createDemoProvider();
  const own = await demo.sendMessageAsync('v-hao', '自己的訊息');
  const attached = await demo.addAttachmentsAsync(own.id, ['demo-file-photo']);
  assert.equal(attached.length, 1);
  assert.equal(attached[0].fileName, '陽明山雲海.jpg');
  const again = await demo.addAttachmentsAsync(own.id, ['demo-file-photo']);
  assert.equal(again.length, 1);
  await assert.rejects(demo.addAttachmentsAsync(own.id, ['demo-file-video']), /尚未通過掃描/);
  await assert.rejects(demo.addAttachmentsAsync(own.id, ['demo-file-missing']), /找不到/);
  await assert.rejects(demo.addAttachmentsAsync('m-missing', ['demo-file-photo']), /找不到訊息/);
  await assert.rejects(demo.addAttachmentsAsync('m-hao-1', ['demo-file-photo']), /只有作者/);
  const sent = await demo.sendMessageAsync('v-hao', '附檔', {
    attachmentFileIds: ['demo-file-photo'],
  });
  const sentAttachments = await demo.listAttachmentsAsync(sent.id);
  assert.equal(sentAttachments.length, 1);
  assert.equal(sentAttachments[0].mediaFileId, 'demo-file-photo');
  await assert.rejects(
    demo.sendMessageAsync('v-hao', '附檔', { attachmentFileIds: ['demo-file-video'] }),
    /尚未通過掃描/,
  );
});

test('demo provider forwards copy content without attachments and cap depth', async () => {
  const demo = createDemoProvider();
  const first = await demo.forwardMessageAsync('m-hao-1', 'v-mei');
  assert.equal(first.conversationId, 'v-mei');
  const record = await demo.getForwardAsync(first.id);
  assert.equal(record.sourceMessageId, 'm-hao-1');
  assert.equal(record.forwardedMessageId, first.id);
  assert.equal(record.targetConversationId, 'v-mei');
  const recordAttachments = await demo.listAttachmentsAsync(first.id);
  assert.equal(recordAttachments.length, 0);
  await assert.rejects(demo.getForwardAsync('m-hao-1'), /找不到轉傳紀錄/);
  await assert.rejects(demo.forwardMessageAsync('m-missing', 'v-mei'), /找不到/);
  await assert.rejects(demo.forwardMessageAsync('m-hao-1', 'v-missing'), /找不到/);
  let current = first;
  const targets = ['v-hao', 'v-mei', 'v-hao'];
  for (const target of targets) {
    current = await demo.forwardMessageAsync(current.id, target);
  }
  await assert.rejects(demo.forwardMessageAsync(current.id, 'v-hao'), /深度/);
});

test('demo provider reactions keep one active reaction and enforce the allowlist', async () => {
  const demo = createDemoProvider();
  const first = await demo.setReactionAsync('m-hao-1', '👍');
  assert.equal(first.reactions.length, 1);
  assert.equal(first.reactions[0].reaction, '👍');
  assert.equal(first.reactions[0].reactedByMe, true);
  const replaced = await demo.setReactionAsync('m-hao-1', '❤️');
  assert.equal(replaced.reactions.length, 1);
  assert.equal(replaced.reactions[0].reaction, '❤️');
  const cleared = await demo.removeReactionAsync('m-hao-1');
  assert.equal(cleared.reactions.length, 0);
  await assert.rejects(demo.setReactionAsync('m-hao-1', '🚀'), /不支援的反應表情/);
  await assert.rejects(demo.setReactionAsync('m-missing', '👍'), /找不到訊息/);
});

test('demo provider pins cap at five and unpin is idempotent', async () => {
  const demo = createDemoProvider();
  const extra1 = await demo.sendMessageAsync('v-hao', '補充一');
  const extra2 = await demo.sendMessageAsync('v-hao', '補充二');
  const extra3 = await demo.sendMessageAsync('v-hao', '補充三');
  await demo.pinMessageAsync('v-hao', 'm-hao-1');
  await demo.pinMessageAsync('v-hao', 'm-hao-2');
  await demo.pinMessageAsync('v-hao', 'm-hao-3');
  await demo.pinMessageAsync('v-hao', extra1.id);
  await demo.pinMessageAsync('v-hao', extra2.id);
  await assert.rejects(demo.pinMessageAsync('v-hao', extra3.id), /上限/);
  const again = await demo.pinMessageAsync('v-hao', 'm-hao-1');
  assert.equal(again.messageId, 'm-hao-1');
  await demo.unpinMessageAsync('v-hao', 'm-hao-1');
  const pinned = await demo.pinMessageAsync('v-hao', extra3.id);
  assert.equal(pinned.messageId, extra3.id);
  await demo.unpinMessageAsync('v-hao', 'm-missing');
  const pins = await demo.listPinsAsync('v-hao');
  assert.equal(pins.length, 5);
  assert.equal(pins[0].messageId, 'm-hao-2');
});

test('messaging 401 clears the token and reports unauthorized', async () => {
  const { provider, store } = await messagingProvider({
    'GET /api/v1/conversations': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  await assert.rejects(provider.listConversationsAsync(), AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('provider listConversationsAsync maps nullable last message fields', async () => {
  const { provider } = await messagingProvider({
    'GET /api/v1/conversations': () => ({
      body: [
        {
          conversation_id: 'c-2',
          type: 'group',
          title: '',
          created_by: 'u-1',
          unread_count: 0,
          created_at: '2026-11-24T00:00:00Z',
          updated_at: '2026-11-24T00:00:00Z',
        },
      ],
    }),
  });
  const conversations = await provider.listConversationsAsync();
  assert.equal(conversations.length, 1);
  assert.equal(conversations[0].type, 'group');
  assert.equal(conversations[0].title, '');
  assert.equal(conversations[0].lastMessageId, '');
  assert.equal(conversations[0].lastMessageAt, null);
  assert.equal(conversations[0].unreadCount, 0);
});

test('provider listMessagesAsync preserves wire order and empty cursor', async () => {
  const older = { ...MESSAGE_WIRE, message_id: 'm-0', sequence_number: 2, content: '早' };
  const newer = { ...MESSAGE_WIRE, message_id: 'm-1', sequence_number: 3, content: '晚' };
  const { provider, calls } = await messagingProvider({
    'GET /api/v1/conversations/c-1/messages': () => ({
      body: { messages: [newer, older] },
    }),
  });
  const page = await provider.listMessagesAsync('c-1', { beforeSeq: 9 });
  assert.deepEqual(
    page.messages.map((message) => message.sequenceNumber),
    [3, 2],
  );
  assert.equal(page.nextCursor, '');
  assert.ok(calls.some((entry) => entry.url.includes('before_seq=9')));
});

test('provider listMessagesAsync clears the token on 401', async () => {
  const { provider, store } = await messagingProvider({
    'GET /api/v1/conversations/c-1/messages': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  await assert.rejects(provider.listMessagesAsync('c-1'), AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('provider exposes the signed-in user id', async () => {
  const { provider } = await messagingProvider();
  assert.equal(provider.getCurrentUserId(), 'u-1');
  const demo = createDemoProvider();
  assert.equal(demo.getCurrentUserId?.(), 'me');
  const signedOut = createApiProvider(
    new EtherApiClient({ baseUrl: 'http://api.test', tokenStore: new MemoryTokenStore() }),
    new MemoryTokenStore(),
  );
  assert.equal(signedOut.getCurrentUserId(), null);
});

test('demo provider listConversationsAsync mirrors snapshot conversations', async () => {
  const provider = createDemoProvider();
  const snapshot = provider.getSnapshot();
  const items = await provider.listConversationsAsync();
  assert.equal(items.length, snapshot.conversations.length);
  assert.deepEqual(
    items.map((item) => item.id),
    snapshot.conversations.map((conversation) => conversation.id),
  );
  assert.deepEqual(
    items.map((item) => item.unreadCount),
    snapshot.conversations.map((conversation) => conversation.unreadCount),
  );
  assert.ok(items.every((item) => item.type === 'direct'));
});

test('demo provider createConversationAsync returns the existing conversation for the same contact', async () => {
  const provider = createDemoProvider();
  const before = await provider.listConversationsAsync();
  const contactId = provider.getSnapshot().conversations[0].contactId;
  const again = await provider.createConversationAsync({ type: 'direct', otherUserId: contactId });
  assert.equal(again.id, before[0].id);
  assert.equal((await provider.listConversationsAsync()).length, before.length);
  const used = new Set(provider.getSnapshot().conversations.map((item) => item.contactId));
  // 含隱藏好友取人選：隱藏只影響清單顯示，不阻擋建立對話；
  // 展示資料中 c-jun 尚無對話，取其識別碼驗證新增路徑。
  const fresh = (await provider.listFriendsAsync(true)).find((friend) => !used.has(friend.id));
  assert.ok(fresh);
  const created = await provider.createConversationAsync({
    type: 'direct',
    otherUserId: fresh.id,
  });
  assert.notEqual(created.id, before[0].id);
  assert.equal((await provider.listConversationsAsync()).length, before.length + 1);
});

test('demo provider sendMessageAsync appends and returns the message', async () => {
  const provider = createDemoProvider();
  const conversationId = provider.getSnapshot().conversations[0].id;
  const before = await provider.listMessagesAsync(conversationId);
  const lastSeq = before.messages.at(-1)?.sequenceNumber ?? 0;
  const sent = await provider.sendMessageAsync(conversationId, '  展示訊息  ');
  assert.equal(sent.content, '展示訊息');
  assert.ok(sent.sequenceNumber > 0);
  assert.equal(sent.status, 'sent');
  const page = await provider.listMessagesAsync(conversationId, { sinceSeq: lastSeq });
  assert.ok(page.messages.some((item) => item.id === sent.id));
  await assert.rejects(provider.sendMessageAsync(conversationId, '   '));
});

test('demo provider listMessagesAsync honors cursor semantics', async () => {
  const provider = createDemoProvider();
  const conversationId = provider.getSnapshot().conversations[0].id;
  const first = await provider.sendMessageAsync(conversationId, '第一則');
  const second = await provider.sendMessageAsync(conversationId, '第二則');
  const third = await provider.sendMessageAsync(conversationId, '第三則');
  const all = await provider.listMessagesAsync(conversationId);
  const mine = all.messages.filter((item) => [first.id, second.id, third.id].includes(item.id));
  assert.deepEqual(
    mine.map((item) => item.sequenceNumber),
    [first.sequenceNumber, second.sequenceNumber, third.sequenceNumber],
  );
  const since = await provider.listMessagesAsync(conversationId, {
    sinceSeq: first.sequenceNumber,
  });
  assert.ok(since.messages.every((item) => item.sequenceNumber > first.sequenceNumber));
  assert.ok(since.messages.some((item) => item.id === third.id));
  const before = await provider.listMessagesAsync(conversationId, {
    beforeSeq: third.sequenceNumber,
  });
  assert.ok(before.messages.every((item) => item.sequenceNumber < third.sequenceNumber));
  assert.ok(before.messages[0].sequenceNumber > before.messages.at(-1).sequenceNumber);
});

test('demo provider edit, recall and delete only apply to own messages', async () => {
  const provider = createDemoProvider();
  const snapshot = provider.getSnapshot();
  const foreign = snapshot.messages.find((item) => item.senderId !== 'me');
  assert.ok(foreign);
  await assert.rejects(provider.editMessageAsync(foreign.id, '改'));
  await assert.rejects(provider.recallMessageAsync(foreign.id));
  const conversationId = snapshot.conversations[0].id;
  const sent = await provider.sendMessageAsync(conversationId, '可編輯');
  const edited = await provider.editMessageAsync(sent.id, '已編輯');
  assert.equal(edited.content, '已編輯');
  assert.equal(edited.editCount, 1);
  const recalled = await provider.recallMessageAsync(sent.id);
  assert.equal(recalled.content, '');
  assert.equal(recalled.status, 'recalled');
  const deleted = await provider.deleteMessageAsync(sent.id);
  assert.equal(deleted.deletedForSelf, true);
});

test('realtime stub never claims a live connection', async () => {
  const client = createDisconnectedRealtimeClient();
  assert.equal(client.isConnected, false);
  await assert.rejects(client.connect(), /尚未接線/);
  await assert.rejects(client.subscribe('c-1'), /尚未接線/);
  await assert.rejects(client.sendTyping('c-1', true), /尚未接線/);
  assert.equal(typeof client.onEvent(() => undefined), 'function');
  client.disconnect();
});

const GROUPS = [
  {
    group_id: 'g-1',
    name: '爬山小隊',
    description: '約爬山',
    avatar_media_id: '',
    conversation_id: 'conv-1',
    created_by: 'u-1',
    is_disbanded: false,
    member_count: 3,
    created_at: '2026-11-20T09:00:00Z',
    updated_at: '2026-11-24T18:30:00Z',
  },
];

const GROUP_MEMBERS = [
  {
    user_id: 'u-1',
    role: 'owner',
    joined_at: '2026-11-20T09:00:00Z',
    invited_by: 'u-1',
    left_at: null,
  },
  {
    user_id: 'u-2',
    role: 'member',
    joined_at: '2026-11-21T09:00:00Z',
    invited_by: 'u-1',
  },
];

const GROUP_INVITATIONS = [
  {
    invitation_id: 'inv-1',
    group_id: 'g-1',
    group_name: '爬山小隊',
    inviter_id: 'u-1',
    invitee_id: 'u-2',
    status: 'pending',
    created_at: '2026-11-24T19:00:00Z',
    updated_at: '2026-11-24T19:00:00Z',
  },
];

const GROUP_ANNOUNCEMENTS = [
  {
    announcement_id: 'ann-1',
    group_id: 'g-1',
    author_id: 'u-1',
    content: '週六七點集合。',
    is_active: true,
    published_at: '2026-11-24T18:30:00Z',
    created_at: '2026-11-24T18:30:00Z',
    updated_at: '2026-11-24T18:30:00Z',
  },
];

const GROUP_SETTINGS = {
  group_id: 'g-1',
  invite_policy: 'admin',
  updated_by: 'u-1',
  updated_at: '2026-11-20T09:00:00Z',
};

function groupRoutes(overrides = {}) {
  return sessionRoutes({
    'GET /api/v1/groups': () => ({ body: GROUPS }),
    'POST /api/v1/groups': ({ init }) => {
      const body = JSON.parse(init.body);
      return {
        status: 201,
        body: {
          ...GROUPS[0],
          group_id: 'g-9',
          name: body.name,
          description: body.description ?? '',
          member_count: 1 + (body.member_ids ?? []).length,
        },
      };
    },
    'GET /api/v1/groups/g-1': () => ({ body: GROUPS[0] }),
    'PATCH /api/v1/groups/g-1': ({ init }) => ({
      body: { ...GROUPS[0], ...JSON.parse(init.body) },
    }),
    'POST /api/v1/groups/g-1/disband': () => ({ body: { status: 'disbanded' } }),
    'POST /api/v1/groups/g-1/transfer': () => ({ body: GROUPS[0] }),
    'GET /api/v1/groups/g-1/members': () => ({ body: GROUP_MEMBERS }),
    'POST /api/v1/groups/g-1/members': ({ init }) => ({
      body: JSON.parse(init.body).member_ids.map((userId) => ({
        user_id: userId,
        role: 'member',
        joined_at: '2026-11-25T10:00:00Z',
        invited_by: 'u-1',
      })),
    }),
    'PATCH /api/v1/groups/g-1/members/u-2': ({ init }) => ({
      body: { ...GROUP_MEMBERS[1], role: JSON.parse(init.body).role },
    }),
    'POST /api/v1/groups/g-1/members/u-2/remove': () => ({ body: { status: 'removed' } }),
    'POST /api/v1/groups/g-1/leave': () => ({ body: { status: 'left' } }),
    'GET /api/v1/groups/g-1/invitations': () => ({ body: GROUP_INVITATIONS }),
    'POST /api/v1/groups/g-1/invitations': ({ init }) => ({
      status: 201,
      body: { ...GROUP_INVITATIONS[0], invitee_id: JSON.parse(init.body).invitee_id },
    }),
    'GET /api/v1/groups/invitations': () => ({ body: GROUP_INVITATIONS }),
    'POST /api/v1/groups/invitations/inv-1/accept': () => ({
      body: { ...GROUP_INVITATIONS[0], status: 'accepted' },
    }),
    'POST /api/v1/groups/invitations/inv-1/decline': () => ({
      body: { ...GROUP_INVITATIONS[0], status: 'declined' },
    }),
    'POST /api/v1/groups/invitations/inv-1/cancel': () => ({
      body: { ...GROUP_INVITATIONS[0], status: 'cancelled' },
    }),
    'GET /api/v1/groups/g-1/announcements': () => ({ body: GROUP_ANNOUNCEMENTS }),
    'POST /api/v1/groups/g-1/announcements': ({ init }) => ({
      status: 201,
      body: {
        ...GROUP_ANNOUNCEMENTS[0],
        announcement_id: 'ann-9',
        content: JSON.parse(init.body).content,
      },
    }),
    'PATCH /api/v1/groups/g-1/announcements/ann-1': ({ init }) => ({
      body: { ...GROUP_ANNOUNCEMENTS[0], content: JSON.parse(init.body).content },
    }),
    'POST /api/v1/groups/g-1/announcements/ann-1/retract': () => ({
      body: {
        ...GROUP_ANNOUNCEMENTS[0],
        is_active: false,
        retracted_at: '2026-11-25T10:00:00Z',
      },
    }),
    'GET /api/v1/groups/g-1/settings': () => ({ body: GROUP_SETTINGS }),
    'PATCH /api/v1/groups/g-1/settings': ({ init }) => ({
      body: { ...GROUP_SETTINGS, invite_policy: JSON.parse(init.body).invite_policy },
    }),
    ...overrides,
  });
}

test('provider listGroupsAsync maps wire groups and my invitations', async () => {
  const { provider, calls } = await readyProvider(groupRoutes());
  const groups = await provider.listGroupsAsync();
  assert.equal(groups.length, 1);
  assert.equal(groups[0].id, 'g-1');
  assert.equal(groups[0].name, '爬山小隊');
  assert.equal(groups[0].memberCount, 3);
  assert.equal(groups[0].isDisbanded, false);
  assert.equal(groups[0].conversationId, 'conv-1');
  const invitations = await provider.listMyGroupInvitationsAsync();
  assert.equal(invitations.length, 1);
  assert.equal(invitations[0].groupName, '爬山小隊');
  assert.equal(invitations[0].status, 'pending');
  assert.notEqual(
    calls.find((item) => item.url === 'http://api.test/api/v1/groups'),
    undefined,
  );
  assert.notEqual(
    calls.find((item) => item.url === 'http://api.test/api/v1/groups/invitations'),
    undefined,
  );
});

test('provider createGroupAsync posts the create body and maps the created group', async () => {
  const { provider, calls } = await readyProvider(groupRoutes());
  const created = await provider.createGroupAsync({
    name: '新群組',
    description: '介紹',
    memberIds: ['u-2', 'u-3'],
  });
  assert.equal(created.id, 'g-9');
  assert.equal(created.name, '新群組');
  assert.equal(created.memberCount, 3);
  const call = calls.find(
    (item) => item.url === 'http://api.test/api/v1/groups' && item.method === 'POST',
  );
  assert.notEqual(call, undefined);
  assert.deepEqual(JSON.parse(call.body), {
    name: '新群組',
    description: '介紹',
    member_ids: ['u-2', 'u-3'],
  });
});

test('provider group member methods map wire members and role changes', async () => {
  const { provider, calls } = await readyProvider(groupRoutes());
  const members = await provider.listGroupMembersAsync('g-1');
  assert.equal(members.length, 2);
  assert.equal(members[0].role, 'owner');
  assert.equal(members[0].leftAt, null);
  assert.equal(members[1].displayName, '');
  const added = await provider.addGroupMembersAsync('g-1', ['u-3']);
  assert.equal(added[0].userId, 'u-3');
  const updated = await provider.setGroupMemberRoleAsync('g-1', 'u-2', 'admin');
  assert.equal(updated.role, 'admin');
  const roleCall = calls.find((item) => item.url.endsWith('/api/v1/groups/g-1/members/u-2'));
  assert.deepEqual(JSON.parse(roleCall.body), { role: 'admin' });
  await provider.removeGroupMemberAsync('g-1', 'u-2');
  await provider.leaveGroupAsync('g-1');
  assert.notEqual(
    calls.find((item) => item.url.endsWith('/members/u-2/remove')),
    undefined,
  );
  assert.notEqual(
    calls.find((item) => item.url.endsWith('/api/v1/groups/g-1/leave')),
    undefined,
  );
});

test('provider group invitation, announcement and settings methods map wire shapes', async () => {
  const { provider, calls } = await readyProvider(groupRoutes());
  const pending = await provider.listGroupInvitationsAsync('g-1');
  assert.equal(pending[0].inviteeId, 'u-2');
  const created = await provider.createGroupInvitationAsync('g-1', 'u-4');
  assert.equal(created.inviteeId, 'u-4');
  const accepted = await provider.respondGroupInvitationAsync('inv-1', true);
  assert.equal(accepted.status, 'accepted');
  const announcements = await provider.listGroupAnnouncementsAsync('g-1');
  assert.equal(announcements[0].isActive, true);
  assert.equal(announcements[0].retractedAt, null);
  const published = await provider.createGroupAnnouncementAsync('g-1', '新公告');
  assert.equal(published.content, '新公告');
  const retracted = await provider.retractGroupAnnouncementAsync('g-1', 'ann-1');
  assert.equal(retracted.isActive, false);
  assert.equal(retracted.retractedAt, '2026-11-25T10:00:00Z');
  const settings = await provider.getGroupSettingsAsync('g-1');
  assert.equal(settings.invitePolicy, 'admin');
  const updated = await provider.updateGroupSettingsAsync('g-1', 'all');
  assert.equal(updated.invitePolicy, 'all');
  const settingsCall = calls.find(
    (item) => item.url.endsWith('/api/v1/groups/g-1/settings') && item.method === 'PATCH',
  );
  assert.deepEqual(JSON.parse(settingsCall.body), { invite_policy: 'all' });
});

test('provider listGroupsAsync clears the token on 401', async () => {
  const { provider, store } = await readyProvider(
    groupRoutes({
      'GET /api/v1/groups': () => ({
        status: 401,
        body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
      }),
    }),
  );
  await assert.rejects(provider.listGroupsAsync(), (error) => error instanceof AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider lists group fixtures and local group actions change only local state', async () => {
  const provider = createDemoProvider();
  const groups = await provider.listGroupsAsync();
  assert.equal(groups.length, DEMO_GROUPS.length);
  assert.equal(groups[0].id, DEMO_GROUPS[0].id);
  const created = await provider.createGroupAsync({
    name: '展示新群組',
    memberIds: ['c-yu'],
  });
  assert.equal(created.memberCount, 2);
  const members = await provider.listGroupMembersAsync(created.id);
  assert.equal(members[0].userId, 'me');
  assert.equal(members[0].role, 'owner');
  const afterCreate = await provider.listGroupsAsync();
  assert.equal(afterCreate.length, DEMO_GROUPS.length + 1);
  const original = DEMO_GROUPS[0];
  const originalMembers = await provider.listGroupMembersAsync(original.id);
  const target = originalMembers.find((item) => item.role === 'member');
  assert.ok(target);
  const promoted = await provider.setGroupMemberRoleAsync(original.id, target.userId, 'admin');
  assert.equal(promoted.role, 'admin');
  await provider.disbandGroupAsync(original.id);
  const afterDisband = await provider.listGroupsAsync();
  assert.equal(
    afterDisband.some((item) => item.id === original.id),
    false,
  );
  assert.equal(afterDisband.length, DEMO_GROUPS.length);
});

test('demo provider group invitations and announcements follow the single-active rule', async () => {
  const provider = createDemoProvider();
  const mine = await provider.listMyGroupInvitationsAsync();
  assert.equal(mine.length, 1);
  assert.equal(mine[0].groupName, '鋼琴同好會');
  const accepted = await provider.respondGroupInvitationAsync(mine[0].id, true);
  assert.equal(accepted.status, 'accepted');
  const joined = await provider.listGroupsAsync();
  assert.equal(
    joined.some((item) => item.id === mine[0].groupId),
    true,
  );
  const groupId = DEMO_GROUPS[0].id;
  const before = await provider.listGroupAnnouncementsAsync(groupId);
  const activeBefore = before.filter((item) => item.isActive);
  assert.equal(activeBefore.length, 1);
  const published = await provider.createGroupAnnouncementAsync(groupId, '新的現行公告');
  assert.equal(published.isActive, true);
  const after = await provider.listGroupAnnouncementsAsync(groupId);
  assert.equal(after.filter((item) => item.isActive).length, 1);
  const previous = after.find((item) => item.id === activeBefore[0].id);
  assert.equal(previous.isActive, false);
  const settings = await provider.getGroupSettingsAsync(groupId);
  assert.equal(settings.invitePolicy, 'admin');
  const updated = await provider.updateGroupSettingsAsync(groupId, 'all');
  assert.equal(updated.invitePolicy, 'all');
});

const MEDIA_FILES = [
  {
    file_id: 'file-1',
    mime_type: 'image/jpeg',
    size_bytes: 184320,
    original_filename: '陽明山雲海.jpg',
    width: 1920,
    height: 1080,
    duration_seconds: 0,
    checksum: '',
    scan_status: 'clean',
    scan_completed_at: '2026-11-24T19:05:00Z',
    created_at: '2026-11-24T19:00:00Z',
    updated_at: '2026-11-24T19:05:00Z',
  },
  {
    file_id: 'file-2',
    mime_type: 'video/mp4',
    size_bytes: 48234496,
    original_filename: '登山口集合.mp4',
    width: 1920,
    height: 1080,
    duration_seconds: 96,
    checksum: '',
    scan_status: 'pending',
    created_at: '2026-11-25T08:12:00Z',
    updated_at: '2026-11-25T08:12:00Z',
  },
];

const MEDIA_UPLOADS = [
  {
    upload_id: 'up-1',
    status: 'uploading',
    storage_key: 'media/u-1/up-1',
    upload_id_external: '',
    uploaded_bytes: 0,
    total_bytes: 48234496,
    part_count: 0,
    created_at: '2026-11-25T08:12:00Z',
    updated_at: '2026-11-25T08:12:00Z',
  },
];

const MEDIA_THUMBNAILS = [
  {
    thumbnail_id: 'thumb-1',
    file_id: 'file-1',
    variant_type: 'thumbnail',
    mime_type: 'image/jpeg',
    size_bytes: 18432,
    width: 320,
    height: 180,
    created_at: '2026-11-24T19:05:00Z',
  },
];

function mediaRoutes(overrides = {}) {
  return sessionRoutes({
    'POST /api/v1/media/uploads': ({ init }) => {
      const body = JSON.parse(init.body);
      return {
        status: 201,
        body: {
          ...MEDIA_UPLOADS[0],
          upload_id: 'up-9',
          status: 'uploading',
          total_bytes: body.total_bytes,
          upload_url: 'https://objects.test/media/u-1/up-9?sig=1',
          upload_url_expires_at: '2026-11-25T08:30:00Z',
        },
      };
    },
    'GET /api/v1/media/uploads': () => ({ body: MEDIA_UPLOADS }),
    'POST /api/v1/media/uploads/up-1/cancel': () => ({
      body: { ...MEDIA_UPLOADS[0], status: 'cancelled' },
    }),
    'POST /api/v1/media/uploads/up-9/upload-url': () => ({
      body: {
        upload_url: 'https://objects.test/media/u-1/up-9?sig=2',
        expires_at: '2026-11-25T08:45:00Z',
      },
    }),
    'POST /api/v1/media/uploads/up-9/progress': ({ init }) => {
      const body = JSON.parse(init.body);
      return {
        body: {
          ...MEDIA_UPLOADS[0],
          upload_id: 'up-9',
          uploaded_bytes: body.uploaded_bytes,
          part_count: body.part_count,
        },
      };
    },
    'POST /api/v1/media/uploads/up-9/complete': ({ init }) => {
      const body = JSON.parse(init.body);
      return {
        body: {
          ...MEDIA_FILES[0],
          file_id: 'file-9',
          original_filename: body.original_filename,
          mime_type: body.mime_type,
          scan_status: 'pending',
        },
      };
    },
    'GET /api/v1/media/files': () => ({ body: MEDIA_FILES }),
    'DELETE /api/v1/media/files/file-1': () => ({ body: { status: 'deleted' } }),
    'GET /api/v1/media/files/file-1/thumbnails': () => ({ body: MEDIA_THUMBNAILS }),
    'POST /api/v1/media/files/file-1/authorize': ({ init }) => ({
      body: {
        allowed: true,
        scan_status: 'clean',
        download_url: 'https://objects.test/media/u-1/file-1?sig=9',
        download_url_expires_at: '2026-11-25T08:30:00Z',
        access_type: JSON.parse(init.body).access_type,
      },
    }),
    'POST /api/v1/media/files/file-2/authorize': () => ({
      body: { allowed: false, scan_status: 'pending' },
    }),
    'POST /api/v1/media/files/file-3/authorize': () => ({
      body: {
        allowed: true,
        scan_status: 'clean',
        download_url: '/api/v1/media/relay/files/file-3',
      },
    }),
    'GET /api/v1/media/relay/files/file-3': () => new Response('relay-bytes', { status: 200 }),
    ...overrides,
  });
}

test('provider initiateMediaUploadAsync posts metadata and maps the session', async () => {
  const { provider, calls } = await readyProvider(mediaRoutes());
  const session = await provider.initiateMediaUploadAsync({
    mimeType: 'image/jpeg',
    totalBytes: 184320,
    originalFilename: '陽明山雲海.jpg',
    width: 0,
    height: 0,
    durationSeconds: 0,
    checksum: '',
  });
  assert.equal(session.id, 'up-9');
  assert.equal(session.status, 'uploading');
  assert.equal(session.totalBytes, 184320);
  assert.equal(session.mediaFileId, '');
  const call = calls.find((item) => item.url.endsWith('/media/uploads') && item.method === 'POST');
  const body = JSON.parse(call.body);
  assert.equal(body.mime_type, 'image/jpeg');
  assert.equal(body.original_filename, '陽明山雲海.jpg');
  assert.equal(body.total_bytes, 184320);
});

test('provider media list, cancel, delete, thumbnail and authorize methods map wire shapes', async () => {
  const { provider, calls } = await readyProvider(mediaRoutes());
  const files = await provider.listMediaFilesAsync();
  assert.equal(files.length, 2);
  assert.equal(files[0].id, 'file-1');
  assert.equal(files[0].scanStatus, 'clean');
  assert.equal(files[0].scanCompletedAt, '2026-11-24T19:05:00Z');
  assert.equal(files[1].scanCompletedAt, null);
  assert.equal(files[1].durationSeconds, 96);
  const uploads = await provider.listMediaUploadsAsync();
  assert.equal(uploads.length, 1);
  assert.equal(uploads[0].storageKey, 'media/u-1/up-1');
  const cancelled = await provider.cancelMediaUploadAsync('up-1');
  assert.equal(cancelled.status, 'cancelled');
  const cancelCall = calls.find((item) => item.url.endsWith('/uploads/up-1/cancel'));
  assert.equal(cancelCall.method, 'POST');
  await provider.deleteMediaFileAsync('file-1');
  const deleteCall = calls.find((item) => item.url.endsWith('/media/files/file-1'));
  assert.equal(deleteCall.method, 'DELETE');
  const thumbnails = await provider.listMediaThumbnailsAsync('file-1');
  assert.equal(thumbnails.length, 1);
  assert.equal(thumbnails[0].variantType, 'thumbnail');
  assert.equal(thumbnails[0].width, 320);
  const allowed = await provider.authorizeMediaAccessAsync('file-1', 'download');
  assert.equal(allowed.allowed, true);
  assert.equal(allowed.scanStatus, 'clean');
  const denied = await provider.authorizeMediaAccessAsync('file-2', 'download');
  assert.equal(denied.allowed, false);
  assert.equal(denied.scanStatus, 'pending');
});

test('provider listMediaFilesAsync clears the token on 401', async () => {
  const { provider, store } = await readyProvider(
    mediaRoutes({
      'GET /api/v1/media/files': () => ({
        status: 401,
        body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
      }),
    }),
  );
  await assert.rejects(provider.listMediaFilesAsync(), AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider media fixtures, session lifecycle and scan gate stay local', async () => {
  const provider = createDemoProvider();
  const files = await provider.listMediaFilesAsync();
  assert.equal(files.length, DEMO_MEDIA_FILES.length);
  assert.equal(files[0].scanStatus, 'clean');
  const uploads = await provider.listMediaUploadsAsync();
  assert.equal(uploads.length, DEMO_MEDIA_UPLOADS.length);
  const created = await provider.initiateMediaUploadAsync({
    mimeType: 'image/png',
    totalBytes: 1024,
    originalFilename: '本機測試.png',
    width: 0,
    height: 0,
    durationSeconds: 0,
    checksum: '',
  });
  assert.equal(created.status, 'uploading');
  assert.equal(created.uploadedBytes, 0);
  assert.equal(created.mediaFileId, '');
  const afterCreate = await provider.listMediaUploadsAsync();
  assert.equal(afterCreate.length, DEMO_MEDIA_UPLOADS.length + 1);
  const cancelled = await provider.cancelMediaUploadAsync(DEMO_MEDIA_UPLOADS[0].id);
  assert.equal(cancelled.status, 'cancelled');
  const clean = await provider.authorizeMediaAccessAsync(DEMO_MEDIA_FILES[0].id, 'download');
  assert.equal(clean.allowed, true);
  const pending = await provider.authorizeMediaAccessAsync(DEMO_MEDIA_FILES[1].id, 'preview');
  assert.equal(pending.allowed, false);
  assert.equal(pending.scanStatus, 'pending');
  const rejected = await provider.authorizeMediaAccessAsync(DEMO_MEDIA_FILES[2].id, 'download');
  assert.equal(rejected.allowed, false);
  assert.equal(rejected.scanStatus, 'rejected');
  await provider.deleteMediaFileAsync(DEMO_MEDIA_FILES[0].id);
  const afterDelete = await provider.listMediaFilesAsync();
  assert.equal(afterDelete.length, DEMO_MEDIA_FILES.length - 1);
  assert.equal(
    afterDelete.some((item) => item.id === DEMO_MEDIA_FILES[0].id),
    false,
  );
  // 展示資料本身不受刪除影響。
  assert.equal(DEMO_MEDIA_FILES.length, 3);
  const thumbnails = await provider.listMediaThumbnailsAsync(DEMO_MEDIA_FILES[0].id);
  assert.equal(thumbnails.length, DEMO_MEDIA_THUMBNAILS.length);
});

test('client searchMessages sends snake_case filters and maps the wire result shape', async () => {
  const seen = [];
  const { client } = makeClient(
    {
      'GET /api/v1/search/messages': ({ url }) => {
        seen.push(url);
        if (url.searchParams.get('q') === '沒有這種詞') {
          return { body: { results: [] } };
        }
        return { body: { results: [MESSAGE_WIRE], next_cursor: 'abc' } };
      },
    },
    'tok-1',
  );
  const page = await client.searchMessages({ q: '提案', conversation_id: 'c-1', limit: 20 });
  assert.equal(seen.length, 1);
  assert.equal(seen[0].searchParams.get('q'), '提案');
  assert.equal(seen[0].searchParams.get('conversation_id'), 'c-1');
  assert.equal(seen[0].searchParams.get('sender_id'), null);
  assert.equal(seen[0].searchParams.get('limit'), '20');
  assert.equal(page.results.length, 1);
  assert.equal(page.results[0].message_id, 'm-1');
  assert.equal(page.results[0].conversation_id, 'c-1');
  assert.equal(page.results[0].sequence_number, 3);
  assert.equal(page.results[0].created_at, '2026-11-25T00:00:00Z');
  assert.equal(page.next_cursor, 'abc');
  const empty = await client.searchMessages({ q: '沒有這種詞' });
  assert.deepEqual(empty.results, []);
  assert.equal(empty.next_cursor, undefined);
});

test('client searchMessages surfaces an unavailable index as ApiError 503', async () => {
  const { client } = makeClient(
    {
      'GET /api/v1/search/messages': () => ({
        status: 503,
        body: { error: { code: 'ETHER_COMMON_UNAVAILABLE', message: '索引不可用。' } },
      }),
    },
    'tok-1',
  );
  const error = await client.searchMessages({ q: '提案' }).then(
    () => null,
    (caught) => caught,
  );
  assert.ok(error instanceof ApiError);
  assert.equal(error.status, 503);
});

test('provider searchMessagesAsync clears the token on 401', async () => {
  const { provider, store } = await messagingProvider({
    'GET /api/v1/search/messages': () => ({
      status: 401,
      body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
    }),
  });
  await assert.rejects(provider.searchMessagesAsync({ query: '提案' }), AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider searchMessagesAsync matches AND terms newest first', async () => {
  const provider = createDemoProvider();
  const page = await provider.searchMessagesAsync({ query: '提案' });
  assert.deepEqual(
    page.results.map((result) => result.id),
    ['m-hao-3', 'm-hao-1'],
  );
  assert.equal(page.nextCursor, '');
  const both = await provider.searchMessagesAsync({ query: '提案 客戶' });
  assert.deepEqual(
    both.results.map((result) => result.id),
    ['m-hao-1'],
  );
  const scoped = await provider.searchMessagesAsync({ query: '提案', conversationId: 'v-mei' });
  assert.equal(scoped.results.length, 0);
  const bySender = await provider.searchMessagesAsync({ query: '提案', senderId: 'me' });
  assert.equal(bySender.results.length, 0);
  // 空關鍵字與後端 q 必填一致：拒絕而非回空結果，面板不送出此類查詢。
  await assert.rejects(provider.searchMessagesAsync({ query: '   ' }), /關鍵字不可為空/);
});

test('demo provider searchMessagesAsync pages by cursor', async () => {
  const provider = createDemoProvider();
  const first = await provider.searchMessagesAsync({ query: '提案', limit: 1 });
  assert.equal(first.results.length, 1);
  assert.equal(first.results[0].id, 'm-hao-3');
  assert.notEqual(first.nextCursor, '');
  const second = await provider.searchMessagesAsync({
    query: '提案',
    limit: 1,
    cursor: first.nextCursor,
  });
  assert.deepEqual(
    second.results.map((result) => result.id),
    ['m-hao-1'],
  );
  assert.equal(second.nextCursor, '');
});

test('demo provider searchMessagesAsync filters by date bounds inclusively', async () => {
  const provider = createDemoProvider();
  const inRange = await provider.searchMessagesAsync({
    query: '週末',
    since: '2026-11-24T00:00:00+08:00',
    until: '2026-11-24T23:59:59+08:00',
  });
  assert.deepEqual(
    inRange.results.map((result) => result.id),
    ['m-yu-1'],
  );
  const outOfRange = await provider.searchMessagesAsync({
    query: '週末',
    since: '2026-11-25T00:00:00+08:00',
  });
  assert.equal(outOfRange.results.length, 0);
});

test('demo provider searchMessagesAsync skips recalled and deleted messages', async () => {
  const provider = createDemoProvider();
  // 「調整」同時命中 m-hao-1（對方）與 m-hao-2（自己）；收回自己的
  // 一則後只剩對方那則。「完整」只命中自己發的 m-ting-2，刪除後無結果。
  const before = await provider.searchMessagesAsync({ query: '調整' });
  assert.deepEqual(
    before.results.map((result) => result.id),
    ['m-hao-2', 'm-hao-1'],
  );
  await provider.recallMessageAsync('m-hao-2');
  const afterRecall = await provider.searchMessagesAsync({ query: '調整' });
  assert.deepEqual(
    afterRecall.results.map((result) => result.id),
    ['m-hao-1'],
  );
  await provider.deleteMessageAsync('m-ting-2');
  const afterDelete = await provider.searchMessagesAsync({ query: '完整' });
  assert.equal(afterDelete.results.length, 0);
});

test('provider initiateMediaUploadAsync maps the issued upload URL', async () => {
  const { provider } = await readyProvider(mediaRoutes());
  const session = await provider.initiateMediaUploadAsync({
    mimeType: 'image/jpeg',
    totalBytes: 184320,
    originalFilename: '陽明山雲海.jpg',
    width: 0,
    height: 0,
    durationSeconds: 0,
    checksum: '',
  });
  assert.equal(session.uploadUrl, 'https://objects.test/media/u-1/up-9?sig=1');
  assert.equal(session.uploadUrlExpiresAt, '2026-11-25T08:30:00Z');
});

test('client putUploadBytes sends no auth header to absolute URLs and auth to relay paths', async () => {
  const { client, calls } = makeClient(
    mediaRoutes({
      'PUT /media/u-1/up-9': () => ({ status: 200, body: {} }),
      'PUT /api/v1/media/relay/uploads/up-9': () => ({ status: 200, body: {} }),
    }),
    'tok-1',
  );
  const data = new Blob(['x'.repeat(12)], { type: 'image/png' });
  await client.putUploadBytes('https://objects.test/media/u-1/up-9?sig=1', data);
  assert.equal(calls[0].url, 'https://objects.test/media/u-1/up-9?sig=1');
  assert.equal(calls[0].method, 'PUT');
  assert.equal(calls[0].headers.Authorization, undefined);
  assert.equal(calls[0].headers['Content-Type'], 'image/png');
  assert.equal(calls[0].body, data);
  await client.putUploadBytes('/api/v1/media/relay/uploads/up-9', data);
  assert.equal(calls[1].url, 'http://api.test/api/v1/media/relay/uploads/up-9');
  assert.equal(calls[1].headers.Authorization, ['Bearer', 'tok-1'].join(' '));
  assert.equal(calls[1].method, 'PUT');
});

test('client putUploadBytes surfaces a 403 presigned failure as an ApiError', async () => {
  const { client } = makeClient(
    mediaRoutes({
      'PUT /media/u-1/up-9': () => ({
        status: 403,
        body: { error: { code: 'ETHER_COMMON_FORBIDDEN', message: '簽名已逾期。' } },
      }),
    }),
    'tok-1',
  );
  await assert.rejects(
    client.putUploadBytes('https://objects.test/media/u-1/up-9?sig=1', new Blob(['x'])),
    (error) => error instanceof ApiError && error.status === 403,
  );
});

test('provider upload flow reports progress and completes without total_bytes', async () => {
  const { provider, calls } = await readyProvider(mediaRoutes());
  const issued = await provider.issueMediaUploadUrlAsync('up-9');
  assert.equal(issued.uploadUrl, 'https://objects.test/media/u-1/up-9?sig=2');
  assert.equal(issued.expiresAt, '2026-11-25T08:45:00Z');
  const progressed = await provider.reportMediaUploadProgressAsync('up-9', 184320, 1);
  assert.equal(progressed.uploadedBytes, 184320);
  assert.equal(progressed.partCount, 1);
  const file = await provider.completeMediaUploadAsync('up-9', {
    mimeType: 'image/jpeg',
    totalBytes: 184320,
    originalFilename: '陽明山雲海.jpg',
    width: 0,
    height: 0,
    durationSeconds: 0,
    checksum: '',
  });
  assert.equal(file.id, 'file-9');
  assert.equal(file.scanStatus, 'pending');
  const completeCall = calls.find((item) => item.url.endsWith('/uploads/up-9/complete'));
  const completeBody = JSON.parse(completeCall.body);
  assert.equal(completeBody.mime_type, 'image/jpeg');
  assert.equal('total_bytes' in completeBody, false);
});

test('provider resolveMediaDownloadAsync distinguishes presigned, relay, forbidden and unavailable', async () => {
  const { provider, calls } = await readyProvider(mediaRoutes());
  const presigned = await provider.resolveMediaDownloadAsync('file-1');
  assert.equal(presigned.status, 'ready-url');
  assert.equal(presigned.url, 'https://objects.test/media/u-1/file-1?sig=9');
  const relay = await provider.resolveMediaDownloadAsync('file-3');
  assert.equal(relay.status, 'ready-blob');
  assert.equal(await relay.blob.text(), 'relay-bytes');
  const relayCall = calls.find((item) => item.url.endsWith('/relay/files/file-3'));
  assert.equal(relayCall.headers.Authorization, ['Bearer', 'tok-1'].join(' '));
  const forbidden = await provider.resolveMediaDownloadAsync('file-2');
  assert.equal(forbidden.status, 'forbidden');
  assert.equal(forbidden.scanStatus, 'pending');
});

test('provider resolveMediaDownloadAsync reports unavailable when allowed but no URL is issued', async () => {
  const { provider } = await readyProvider(
    mediaRoutes({
      'POST /api/v1/media/files/file-1/authorize': () => ({
        body: { allowed: true, scan_status: 'clean' },
      }),
    }),
  );
  const resolved = await provider.resolveMediaDownloadAsync('file-1');
  assert.equal(resolved.status, 'unavailable');
});

test('provider completeMediaUploadAsync clears the token on 401', async () => {
  const { provider, store } = await readyProvider(
    mediaRoutes({
      'POST /api/v1/media/uploads/up-9/complete': () => ({
        status: 401,
        body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
      }),
    }),
  );
  await assert.rejects(
    provider.completeMediaUploadAsync('up-9', {
      mimeType: 'image/jpeg',
      totalBytes: 12,
      originalFilename: 'a.jpg',
      width: 0,
      height: 0,
      durationSeconds: 0,
      checksum: '',
    }),
    AuthError,
  );
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider byte flow uploads, completes and resolves downloads locally', async () => {
  const provider = createDemoProvider();
  const created = await provider.initiateMediaUploadAsync({
    mimeType: 'image/png',
    totalBytes: 12,
    originalFilename: '本機位元組.png',
    width: 0,
    height: 0,
    durationSeconds: 0,
    checksum: '',
  });
  assert.match(created.uploadUrl, /^demo:\/\/uploads\//);
  const bytes = new Blob(['x'.repeat(12)], { type: 'image/png' });
  await provider.putMediaUploadBytesAsync(created.uploadUrl, bytes);
  const progressed = await provider.reportMediaUploadProgressAsync(created.id, 12, 1);
  assert.equal(progressed.uploadedBytes, 12);
  const file = await provider.completeMediaUploadAsync(created.id, {
    mimeType: 'image/png',
    totalBytes: 12,
    originalFilename: '本機位元組.png',
    width: 0,
    height: 0,
    durationSeconds: 0,
    checksum: '',
  });
  assert.equal(file.scanStatus, 'pending');
  assert.equal(file.sizeBytes, 12);
  // 完成後重複完成回傳同一檔案（冪等），不重建檔案列。
  const again = await provider.completeMediaUploadAsync(created.id, {
    mimeType: 'image/png',
    totalBytes: 12,
    originalFilename: '本機位元組.png',
    width: 0,
    height: 0,
    durationSeconds: 0,
    checksum: '',
  });
  assert.equal(again.id, file.id);
  const files = await provider.listMediaFilesAsync();
  assert.equal(files.filter((item) => item.id === file.id).length, 1);
});

test('demo provider refuses completion without bytes and rejects wrong-size puts', async () => {
  const provider = createDemoProvider();
  const created = await provider.initiateMediaUploadAsync({
    mimeType: 'image/png',
    totalBytes: 12,
    originalFilename: '未傳.png',
    width: 0,
    height: 0,
    durationSeconds: 0,
    checksum: '',
  });
  await assert.rejects(
    provider.completeMediaUploadAsync(created.id, {
      mimeType: 'image/png',
      totalBytes: 12,
      originalFilename: '未傳.png',
      width: 0,
      height: 0,
      durationSeconds: 0,
      checksum: '',
    }),
    /尚未上傳/,
  );
  await assert.rejects(
    provider.putMediaUploadBytesAsync(created.uploadUrl, new Blob(['xx'])),
    /大小須等於宣告總量/,
  );
  await assert.rejects(
    provider.putMediaUploadBytesAsync('https://objects.test/nope', new Blob(['x'.repeat(12)])),
    /找不到進行中/,
  );
});

test('demo provider download resolution honors the scan gate and missing bytes', async () => {
  const provider = createDemoProvider();
  const files = await provider.listMediaFilesAsync();
  const clean = files.find((item) => item.scanStatus === 'clean');
  const pending = files.find((item) => item.scanStatus === 'pending');
  const cleanResolved = await provider.resolveMediaDownloadAsync(clean.id);
  assert.equal(cleanResolved.status, 'unavailable');
  const pendingResolved = await provider.resolveMediaDownloadAsync(pending.id);
  assert.equal(pendingResolved.status, 'forbidden');
  assert.equal(pendingResolved.scanStatus, 'pending');
});

const POST_WIRE = {
  post_id: 'p-1',
  author_id: 'u-2',
  content: '週六下午三點，媽媽家吃飯。',
  visibility: 'public',
  edited_at: null,
  media: [{ media_file_id: 'f-1', position: 0, caption: '' }],
  like_count: 2,
  comment_count: 3,
  liked_by_me: false,
  created_at: '2026-11-25T10:30:00Z',
  updated_at: '2026-11-25T10:30:00Z',
};

const POST_COMMENT_WIRE = {
  comment_id: 'cm-1',
  post_id: 'p-1',
  author_id: 'u-2',
  parent_comment_id: '',
  content: '收到，週六見！',
  is_deleted: false,
  edited_at: null,
  created_at: '2026-11-25T10:35:00Z',
  updated_at: '2026-11-25T10:35:00Z',
};

function timelineRoutes(overrides = {}) {
  return identityRoutes({
    'POST /api/v1/posts': ({ init }) => ({
      status: 201,
      body: {
        ...POST_WIRE,
        content: JSON.parse(init.body).content,
        visibility: JSON.parse(init.body).visibility,
      },
    }),
    'GET /api/v1/posts/p-1': () => ({ body: POST_WIRE }),
    'PATCH /api/v1/posts/p-1': ({ init }) => ({
      body: { ...POST_WIRE, ...JSON.parse(init.body), edited_at: '2026-11-25T11:00:00Z' },
    }),
    'POST /api/v1/posts/p-1/delete': () => ({ body: { status: 'deleted' } }),
    'GET /api/v1/posts': () => ({
      body: { posts: [POST_WIRE], next_cursor: 'cursor-next' },
    }),
    'GET /api/v1/timeline': () => ({
      body: { posts: [POST_WIRE], next_cursor: 'cursor-next' },
    }),
    'GET /api/v1/posts/p-1/media': () => ({ body: POST_WIRE.media }),
    'POST /api/v1/posts/p-1/likes': () => ({
      body: { post_id: 'p-1', liked: true, like_count: 3 },
    }),
    'DELETE /api/v1/posts/p-1/likes': () => ({
      body: { post_id: 'p-1', liked: false, like_count: 2 },
    }),
    'POST /api/v1/posts/p-1/comments': ({ init }) => ({
      status: 201,
      body: { ...POST_COMMENT_WIRE, content: JSON.parse(init.body).content },
    }),
    'GET /api/v1/posts/p-1/comments': () => ({
      body: { comments: [POST_COMMENT_WIRE], next_cursor: '' },
    }),
    'POST /api/v1/comments/cm-1/delete': () => ({ body: { status: 'deleted' } }),
    ...overrides,
  });
}

test('client sends cursor and limit params for feed and author posts', async () => {
  const { client, calls } = makeClient(timelineRoutes(), 'tok-1');
  await client.listFeed({ cursor: 'abc', limit: 10 });
  assert.ok(calls[0].url.includes('/timeline?'));
  assert.ok(calls[0].url.includes('cursor=abc'));
  assert.ok(calls[0].url.includes('limit=10'));
  await client.listAuthorPosts('u-2', { limit: 5 });
  assert.ok(calls[1].url.includes('/posts?'));
  assert.ok(calls[1].url.includes('author_id=u-2'));
  assert.ok(calls[1].url.includes('limit=5'));
  assert.ok(!calls[1].url.includes('cursor='));
});

test('provider listFeedAsync maps the wire page and listAuthorPostsAsync maps posts', async () => {
  const { provider } = await readyProvider(timelineRoutes());
  const feed = await provider.listFeedAsync();
  assert.equal(feed.posts.length, 1);
  assert.equal(feed.posts[0].id, 'p-1');
  assert.equal(feed.posts[0].authorId, 'u-2');
  assert.equal(feed.posts[0].visibility, 'public');
  assert.equal(feed.posts[0].likeCount, 2);
  assert.equal(feed.posts[0].commentCount, 3);
  assert.equal(feed.posts[0].likedByMe, false);
  assert.equal(feed.posts[0].media[0].mediaFileId, 'f-1');
  assert.equal(feed.nextCursor, 'cursor-next');
  const mine = await provider.listAuthorPostsAsync('u-2');
  assert.equal(mine.posts[0].id, 'p-1');
});

test('provider create/update/delete post posts the right bodies and maps the post', async () => {
  const { provider, calls } = await readyProvider(timelineRoutes());
  const created = await provider.createPostAsync({
    content: '新貼文',
    visibility: 'friends',
    mediaFileIds: ['f-1'],
  });
  assert.equal(created.content, '新貼文');
  assert.equal(created.visibility, 'friends');
  const createCall = calls.find(
    (item) => item.method === 'POST' && item.url.endsWith('/api/v1/posts'),
  );
  assert.equal(JSON.parse(createCall.body).media_file_ids[0], 'f-1');
  const updated = await provider.updatePostAsync('p-1', { content: '改過' });
  assert.equal(updated.content, '改過');
  const patchCall = calls.find((item) => item.method === 'PATCH');
  assert.equal(JSON.parse(patchCall.body).visibility, undefined);
  await provider.deletePostAsync('p-1');
  assert.notEqual(
    calls.find((item) => item.url.endsWith('/api/v1/posts/p-1/delete')),
    undefined,
  );
});

test('provider like/unlike and comment methods map wire results', async () => {
  const { provider, calls } = await readyProvider(timelineRoutes());
  const liked = await provider.likePostAsync('p-1');
  assert.deepEqual(liked, { postId: 'p-1', liked: true, likeCount: 3 });
  const unliked = await provider.unlikePostAsync('p-1');
  assert.deepEqual(unliked, { postId: 'p-1', liked: false, likeCount: 2 });
  const comment = await provider.createPostCommentAsync('p-1', '收到', 'cm-0');
  assert.equal(comment.id, 'cm-1');
  assert.equal(comment.parentCommentId, '');
  const createCall = calls.find((item) => item.url.endsWith('/api/v1/posts/p-1/comments'));
  assert.equal(JSON.parse(createCall.body).parent_comment_id, 'cm-0');
  const page = await provider.listPostCommentsAsync('p-1');
  assert.equal(page.comments.length, 1);
  assert.equal(page.comments[0].content, '收到，週六見！');
  const media = await provider.listPostMediaAsync('p-1');
  assert.equal(media[0].mediaFileId, 'f-1');
  await provider.deletePostCommentAsync('cm-1');
  assert.notEqual(
    calls.find((item) => item.url.endsWith('/api/v1/comments/cm-1/delete')),
    undefined,
  );
});

test('provider listFeedAsync clears the token on 401', async () => {
  const { provider, store } = await readyProvider(
    timelineRoutes({
      'GET /api/v1/timeline': () => ({
        status: 401,
        body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
      }),
    }),
  );
  await assert.rejects(provider.listFeedAsync(), (error) => error instanceof AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider feed hides blocked and only-me posts and paginates with cursors', async () => {
  const provider = createDemoProvider();
  const first = await provider.listFeedAsync({ limit: 1 });
  assert.equal(first.posts.length, 1);
  assert.equal(first.posts[0].id, 'demo-post-1');
  assert.notEqual(first.nextCursor, '');
  const second = await provider.listFeedAsync({ cursor: first.nextCursor, limit: 1 });
  assert.equal(second.posts[0].id, 'demo-post-2');
  const third = await provider.listFeedAsync({ cursor: second.nextCursor, limit: 1 });
  assert.equal(third.posts[0].id, 'demo-post-3');
  assert.equal(third.nextCursor, '');
  const all = await provider.listFeedAsync();
  assert.deepEqual(
    all.posts.map((post) => post.id),
    ['demo-post-1', 'demo-post-2', 'demo-post-3'],
  );
  await assert.rejects(provider.listFeedAsync({ cursor: '!!!' }), /游標格式錯誤/);
});

test('demo provider post guards mirror the backend: author-only edit/delete, media gate, limits', async () => {
  const provider = createDemoProvider();
  await assert.rejects(provider.createPostAsync({ content: '', mediaFileIds: [] }), /不得皆為空/);
  await assert.rejects(provider.createPostAsync({ content: 'x'.repeat(2001) }), /不得超過 2000 字/);
  await assert.rejects(
    provider.createPostAsync({ content: '', mediaFileIds: ['demo-file-video'] }),
    /尚未通過掃描/,
  );
  const mediaOnly = await provider.createPostAsync({
    content: '',
    mediaFileIds: ['demo-file-photo'],
  });
  assert.equal(mediaOnly.media.length, 1);
  assert.equal(mediaOnly.visibility, 'friends');
  await assert.rejects(provider.updatePostAsync('demo-post-1', { content: '改' }), /找不到貼文/);
  await assert.rejects(provider.deletePostAsync('demo-post-1'), /找不到貼文/);
  const edited = await provider.updatePostAsync('demo-post-3', { content: '改過的內容' });
  assert.equal(edited.content, '改過的內容');
  assert.notEqual(edited.editedAt, null);
  const textOnly = await provider.createPostAsync({ content: '純文字貼文' });
  await assert.rejects(provider.updatePostAsync(textOnly.id, { content: '' }), /不得皆為空/);
  await provider.deletePostAsync('demo-post-3');
  await assert.rejects(provider.getPostAsync('demo-post-3'), /找不到貼文/);
});

test('demo provider likes and comments follow live counts and soft-delete display', async () => {
  const provider = createDemoProvider();
  const liked = await provider.likePostAsync('demo-post-1');
  assert.equal(liked.likeCount, 3);
  assert.equal(liked.liked, true);
  const unliked = await provider.unlikePostAsync('demo-post-1');
  assert.equal(unliked.likeCount, 2);
  const post = await provider.getPostAsync('demo-post-1');
  assert.equal(post.likeCount, 2);
  assert.equal(post.commentCount, 2);
  const reply = await provider.createPostCommentAsync('demo-post-2', '回覆內容', 'demo-comment-4');
  assert.equal(reply.parentCommentId, 'demo-comment-4');
  await assert.rejects(
    provider.createPostCommentAsync('demo-post-2', '錯', 'demo-comment-1'),
    /不屬於此貼文/,
  );
  await assert.rejects(provider.createPostCommentAsync('demo-post-2', '   '), /不得為空/);
  const comments = await provider.listPostCommentsAsync('demo-post-1');
  const deleted = comments.comments.find((comment) => comment.id === 'demo-comment-3');
  assert.equal(deleted.content, '');
  assert.equal(deleted.isDeleted, true);
  await assert.rejects(provider.deletePostCommentAsync('demo-comment-1'), /找不到留言/);
  await provider.deletePostCommentAsync('demo-comment-4');
  const after = await provider.getPostAsync('demo-post-2');
  assert.equal(after.commentCount, 1);
});

const CALL_WIRE = {
  call_id: 'call-1',
  conversation_id: 'c-1',
  initiator_id: 'u-1',
  call_type: 'voice',
  status: 'ringing',
  started_at: null,
  ended_at: null,
  duration_seconds: null,
  ended_by: '',
  end_reason: '',
  participant_count: 2,
  created_at: '2026-11-25T12:50:00Z',
  updated_at: '2026-11-25T12:50:00Z',
};

const CALL_PARTICIPANT_WIRE = {
  user_id: 'u-1',
  joined_at: '2026-11-25T12:50:00Z',
  left_at: null,
  result: 'joined',
  created_at: '2026-11-25T12:50:00Z',
};

const CALL_EVENT_WIRE = {
  event_id: 'ev-1',
  user_id: 'u-1',
  event_type: 'initiated',
  occurred_at: '2026-11-25T12:50:00Z',
};

function callRoutes(overrides = {}) {
  return identityRoutes({
    'POST /api/v1/calls': ({ init }) => ({
      status: 201,
      body: { ...CALL_WIRE, ...JSON.parse(init.body) },
    }),
    'GET /api/v1/calls': () => ({ body: { calls: [CALL_WIRE], next_cursor: 'cursor-next' } }),
    'GET /api/v1/calls/call-1': () => ({ body: CALL_WIRE }),
    'POST /api/v1/calls/call-1/answer': () => ({
      body: { ...CALL_WIRE, status: 'active', started_at: '2026-11-25T12:51:00Z' },
    }),
    'POST /api/v1/calls/call-1/decline': () => ({
      body: { ...CALL_WIRE, status: 'declined', end_reason: 'declined' },
    }),
    'POST /api/v1/calls/call-1/busy': () => ({
      body: { ...CALL_WIRE, status: 'busy', end_reason: 'busy' },
    }),
    'POST /api/v1/calls/call-1/end': () => ({
      body: {
        ...CALL_WIRE,
        status: 'ended',
        end_reason: 'hangup',
        duration_seconds: 90,
        ended_at: '2026-11-25T12:53:00Z',
      },
    }),
    'GET /api/v1/calls/call-1/participants': () => ({ body: [CALL_PARTICIPANT_WIRE] }),
    'GET /api/v1/calls/call-1/events': () => ({ body: [CALL_EVENT_WIRE] }),
    ...overrides,
  });
}

test('client call methods hit the calls endpoints with cursor and limit params', async () => {
  const { client, calls } = makeClient(callRoutes(), 'tok-1');
  await client.listCalls({ cursor: 'abc', limit: 10 });
  assert.ok(calls[0].url.includes('/api/v1/calls?'));
  assert.ok(calls[0].url.includes('cursor=abc'));
  assert.ok(calls[0].url.includes('limit=10'));
  await client.getCall('call-1');
  await client.answerCall('call-1');
  await client.declineCall('call-1');
  await client.busyCall('call-1');
  await client.endCall('call-1');
  await client.listCallParticipants('call-1');
  await client.listCallEvents('call-1');
  const urls = calls.map((entry) => `${entry.method} ${entry.url}`);
  assert.ok(urls.some((url) => url.endsWith('GET http://api.test/api/v1/calls/call-1')));
  assert.ok(urls.some((url) => url.endsWith('/calls/call-1/answer')));
  assert.ok(urls.some((url) => url.endsWith('/calls/call-1/decline')));
  assert.ok(urls.some((url) => url.endsWith('/calls/call-1/busy')));
  assert.ok(urls.some((url) => url.endsWith('/calls/call-1/end')));
  assert.ok(urls.some((url) => url.endsWith('/calls/call-1/participants')));
  assert.ok(urls.some((url) => url.endsWith('/calls/call-1/events')));
});

test('provider call methods map wire shapes and send exactly one scope id', async () => {
  const { provider, calls } = await readyProvider(callRoutes());
  const page = await provider.listCallsAsync();
  assert.equal(page.calls.length, 1);
  assert.equal(page.calls[0].id, 'call-1');
  assert.equal(page.calls[0].conversationId, 'c-1');
  assert.equal(page.calls[0].groupId, '');
  assert.equal(page.calls[0].status, 'ringing');
  assert.equal(page.calls[0].startedAt, null);
  assert.equal(page.calls[0].durationSeconds, null);
  assert.equal(page.nextCursor, 'cursor-next');
  const found = await provider.getCallAsync('call-1');
  assert.equal(found.callType, 'voice');
  const created = await provider.initiateCallAsync({ conversationId: 'c-1', callType: 'video' });
  assert.equal(created.callType, 'video');
  const createCall = calls.find(
    (item) => item.method === 'POST' && item.url.endsWith('/api/v1/calls'),
  );
  assert.deepEqual(JSON.parse(createCall.body), { conversation_id: 'c-1', call_type: 'video' });
  await provider.initiateCallAsync({ groupId: 'g-1', callType: 'voice' });
  const groupCall = calls.filter(
    (item) => item.method === 'POST' && item.url.endsWith('/api/v1/calls'),
  )[1];
  assert.deepEqual(JSON.parse(groupCall.body), { group_id: 'g-1', call_type: 'voice' });
  const answered = await provider.answerCallAsync('call-1');
  assert.equal(answered.status, 'active');
  assert.equal(answered.startedAt, '2026-11-25T12:51:00Z');
  const declined = await provider.declineCallAsync('call-1');
  assert.equal(declined.status, 'declined');
  assert.equal(declined.endReason, 'declined');
  const busy = await provider.busyCallAsync('call-1');
  assert.equal(busy.status, 'busy');
  const ended = await provider.endCallAsync('call-1');
  assert.equal(ended.status, 'ended');
  assert.equal(ended.durationSeconds, 90);
  const participants = await provider.listCallParticipantsAsync('call-1');
  assert.equal(participants[0].userId, 'u-1');
  assert.equal(participants[0].result, 'joined');
  const events = await provider.listCallEventsAsync('call-1');
  assert.equal(events[0].id, 'ev-1');
  assert.equal(events[0].eventType, 'initiated');
  assert.equal('payload' in events[0], false);
});

test('provider listCallsAsync clears the token on 401', async () => {
  const { provider, store } = await readyProvider(
    callRoutes({
      'GET /api/v1/calls': () => ({
        status: 401,
        body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
      }),
    }),
  );
  await assert.rejects(provider.listCallsAsync(), (error) => error instanceof AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider call list paginates with cursors and hides non-participant calls', async () => {
  const provider = createDemoProvider();
  const first = await provider.listCallsAsync({ limit: 1 });
  assert.equal(first.calls.length, 1);
  assert.equal(first.calls[0].id, 'demo-call-4');
  assert.notEqual(first.nextCursor, '');
  const second = await provider.listCallsAsync({ cursor: first.nextCursor, limit: 1 });
  assert.equal(second.calls[0].id, 'demo-call-1');
  const all = await provider.listCallsAsync();
  assert.deepEqual(
    all.calls.map((call) => call.id),
    ['demo-call-4', 'demo-call-1', 'demo-call-2', 'demo-call-3'],
  );
  await assert.rejects(provider.listCallsAsync({ cursor: '!!!' }), /游標格式錯誤/);
  await assert.rejects(provider.getCallAsync('no-such-call'), /找不到通話/);
});

test('demo provider call state machine mirrors the backend: answer, decline, busy, end', async () => {
  const answering = createDemoProvider();
  const answered = await answering.answerCallAsync('demo-call-4');
  assert.equal(answered.status, 'active');
  assert.notEqual(answered.startedAt, null);
  const answeredEvents = await answering.listCallEventsAsync('demo-call-4');
  assert.equal(answeredEvents[answeredEvents.length - 1].eventType, 'answered');
  assert.equal('payload' in answeredEvents[0], false);
  const ended = await answering.endCallAsync('demo-call-4');
  assert.equal(ended.status, 'ended');
  assert.equal(ended.endReason, 'hangup');
  assert.notEqual(ended.durationSeconds, null);
  await assert.rejects(answering.answerCallAsync('demo-call-4'), /已不在可接聽狀態/);

  const declining = createDemoProvider();
  const declined = await declining.declineCallAsync('demo-call-4');
  assert.equal(declined.status, 'declined');
  assert.equal(declined.endReason, 'declined');
  assert.equal(declined.durationSeconds, null);

  const busying = createDemoProvider();
  const busy = await busying.busyCallAsync('demo-call-4');
  assert.equal(busy.status, 'busy');
  assert.equal(busy.endReason, 'busy');
});

test('demo provider call guards mirror the backend: scope, blocks, duplicate, initiator', async () => {
  const provider = createDemoProvider();
  await assert.rejects(provider.initiateCallAsync({ callType: 'voice' }), /其中之一/);
  await assert.rejects(
    provider.initiateCallAsync({ conversationId: 'v-hao', groupId: 'g-1', callType: 'voice' }),
    /其中之一/,
  );
  await assert.rejects(
    provider.initiateCallAsync({ conversationId: 'v-hao', callType: 'screen' }),
    /voice 或 video/,
  );
  await assert.rejects(
    provider.initiateCallAsync({ conversationId: 'v-wei', callType: 'voice' }),
    /找不到通話/,
  );
  await assert.rejects(
    provider.initiateCallAsync({ conversationId: 'v-mei', callType: 'voice' }),
    /已有進行中/,
  );
  await assert.rejects(provider.answerCallAsync('demo-call-1'), /已不在可接聽狀態/);
  const initiated = await provider.initiateCallAsync({
    conversationId: 'v-hao',
    callType: 'voice',
  });
  assert.equal(initiated.status, 'ringing');
  assert.equal(initiated.participantCount, 2);
  await assert.rejects(provider.answerCallAsync(initiated.id), /已回應/);
  await assert.rejects(provider.declineCallAsync(initiated.id), /已回應/);
  const missed = await provider.endCallAsync(initiated.id);
  assert.equal(missed.status, 'missed');
  assert.equal(missed.endReason, 'missed');
  assert.equal(missed.durationSeconds, null);
  const participants = await provider.listCallParticipantsAsync(initiated.id);
  assert.equal(participants.find((p) => p.userId === 'c-hao').result, 'missed');
  await assert.rejects(provider.endCallAsync(initiated.id), /已結束/);
  const groupCall = await provider.initiateCallAsync({
    groupId: 'demo-group-reading',
    callType: 'video',
  });
  assert.equal(groupCall.groupId, 'demo-group-reading');
  assert.equal(groupCall.participantCount, 3);
});

const STICKER_PACK_WIRE = {
  pack_id: 'pack-1',
  creator_id: 'u-1',
  name: '山間小日子',
  description: '山裡的日常。',
  category: 'general',
  cover_media_id: '',
  status: 'published',
  is_free: true,
  sticker_count: 2,
  favorited_by_me: false,
  created_at: '2026-11-25T09:00:00Z',
  updated_at: '2026-11-25T09:00:00Z',
};

const STICKER_WIRE = {
  sticker_id: 'sticker-1',
  pack_id: 'pack-1',
  pack_name: '山間小日子',
  media_file_id: 'media-1',
  name: '咖啡時間',
  position: 0,
  is_animated: false,
  status: 'active',
  created_at: '2026-11-25T09:00:00Z',
  updated_at: '2026-11-25T09:00:00Z',
};

function stickerRoutes(overrides = {}) {
  return identityRoutes({
    'POST /api/v1/sticker-packs': ({ init }) => ({
      status: 201,
      body: { ...STICKER_PACK_WIRE, ...JSON.parse(init.body) },
    }),
    'GET /api/v1/sticker-packs': () => ({
      body: { packs: [STICKER_PACK_WIRE], next_cursor: 'cursor-next' },
    }),
    'GET /api/v1/sticker-packs/pack-1': () => ({ body: STICKER_PACK_WIRE }),
    'PATCH /api/v1/sticker-packs/pack-1': ({ init }) => ({
      body: { ...STICKER_PACK_WIRE, ...JSON.parse(init.body) },
    }),
    'POST /api/v1/sticker-packs/pack-1/delete': () => ({ body: { status: 'archived' } }),
    'POST /api/v1/sticker-packs/pack-1/favorite': () => ({
      body: { ...STICKER_PACK_WIRE, favorited_by_me: true },
    }),
    'DELETE /api/v1/sticker-packs/pack-1/favorite': () => ({
      body: { ...STICKER_PACK_WIRE, favorited_by_me: false },
    }),
    'GET /api/v1/stickers/favorites': () => ({
      body: { packs: [STICKER_PACK_WIRE], next_cursor: '' },
    }),
    'GET /api/v1/sticker-packs/pack-1/stickers': () => ({ body: [STICKER_WIRE] }),
    'POST /api/v1/sticker-packs/pack-1/stickers': ({ init }) => ({
      status: 201,
      body: { ...STICKER_WIRE, ...JSON.parse(init.body) },
    }),
    'POST /api/v1/stickers/sticker-1/delete': () => ({ body: { status: 'removed' } }),
    'GET /api/v1/stickers/search': () => ({
      body: { stickers: [STICKER_WIRE], next_cursor: 'cursor-next' },
    }),
    'GET /api/v1/stickers/recent': () => ({
      body: { stickers: [STICKER_WIRE], next_cursor: '' },
    }),
    'POST /api/v1/stickers/sticker-1/use': () => ({ body: STICKER_WIRE }),
    ...overrides,
  });
}

test('client sticker methods hit the sticker endpoints with cursor and filter params', async () => {
  const { client, calls } = makeClient(stickerRoutes(), 'tok-1');
  await client.listStickerPacks({
    creatorId: 'u-1',
    category: 'animals',
    q: '山',
    cursor: 'abc',
    limit: 10,
  });
  assert.ok(calls[0].url.includes('/api/v1/sticker-packs?'));
  assert.ok(calls[0].url.includes('creator_id=u-1'));
  assert.ok(calls[0].url.includes('category=animals'));
  assert.ok(calls[0].url.includes('q=%E5%B1%B1'));
  assert.ok(calls[0].url.includes('cursor=abc'));
  assert.ok(calls[0].url.includes('limit=10'));
  await client.createStickerPack({
    name: '新包',
    description: '',
    category: 'food',
    cover_media_id: '',
    is_free: true,
  });
  await client.getStickerPack('pack-1');
  await client.updateStickerPack('pack-1', { name: '改名', status: 'unlisted' });
  await client.deleteStickerPack('pack-1');
  await client.favoriteStickerPack('pack-1');
  await client.unfavoriteStickerPack('pack-1');
  await client.listFavoriteStickerPacks({ cursor: 'c', limit: 5 });
  await client.listPackStickers('pack-1');
  await client.addSticker('pack-1', {
    media_file_id: 'media-1',
    name: '新貼圖',
    is_animated: false,
  });
  await client.removeSticker('sticker-1');
  await client.searchStickers('咖啡', { category: 'general', cursor: 'c', limit: 5 });
  await client.listRecentStickers({ cursor: 'c', limit: 5 });
  await client.recordStickerUse('sticker-1');
  const urls = calls.map((entry) => `${entry.method} ${entry.url}`);
  assert.ok(urls.some((url) => url.endsWith('POST http://api.test/api/v1/sticker-packs')));
  assert.ok(urls.some((url) => url.endsWith('GET http://api.test/api/v1/sticker-packs/pack-1')));
  assert.ok(urls.some((url) => url.endsWith('PATCH http://api.test/api/v1/sticker-packs/pack-1')));
  assert.ok(
    urls.some((url) => url.endsWith('POST http://api.test/api/v1/sticker-packs/pack-1/delete')),
  );
  assert.ok(urls.some((url) => url.endsWith('/sticker-packs/pack-1/favorite')));
  assert.ok(
    urls.some((url) =>
      url.endsWith('GET http://api.test/api/v1/stickers/favorites?cursor=c&limit=5'),
    ),
  );
  assert.ok(urls.some((url) => url.endsWith('/sticker-packs/pack-1/stickers')));
  assert.ok(
    urls.some((url) => url.endsWith('POST http://api.test/api/v1/stickers/sticker-1/delete')),
  );
  assert.ok(urls.some((url) => url.includes('/api/v1/stickers/search?q=%E5%92%96%E5%95%A1')));
  assert.ok(urls.some((url) => url.includes('/api/v1/stickers/recent?')));
  assert.ok(urls.some((url) => url.endsWith('/stickers/sticker-1/use')));
  const createCall = calls.find(
    (item) => item.method === 'POST' && item.url.endsWith('/api/v1/sticker-packs'),
  );
  assert.deepEqual(JSON.parse(createCall.body), {
    name: '新包',
    description: '',
    category: 'food',
    cover_media_id: '',
    is_free: true,
  });
  const searchCall = calls.find((item) => item.url.includes('/api/v1/stickers/search?'));
  assert.ok(searchCall.url.includes('category=general'));
});

test('client searchStickers rejects an empty query before any request', async () => {
  const { client, calls } = makeClient(stickerRoutes(), 'tok-1');
  await assert.rejects(client.searchStickers('  '), /搜尋關鍵字不得為空/);
  assert.equal(calls.length, 0);
});

test('provider sticker methods map wire shapes and snake_case bodies', async () => {
  const { provider, calls } = await readyProvider(stickerRoutes());
  const page = await provider.listStickerPacksAsync({ limit: 10 });
  assert.equal(page.packs.length, 1);
  assert.equal(page.packs[0].id, 'pack-1');
  assert.equal(page.packs[0].creatorId, 'u-1');
  assert.equal(page.packs[0].category, 'general');
  assert.equal(page.packs[0].coverMediaId, '');
  assert.equal(page.packs[0].isFree, true);
  assert.equal(page.packs[0].stickerCount, 2);
  assert.equal(page.packs[0].favoritedByMe, false);
  assert.equal(page.nextCursor, 'cursor-next');
  const found = await provider.getStickerPackAsync('pack-1');
  assert.equal(found.name, '山間小日子');
  const created = await provider.createStickerPackAsync({
    name: '新包',
    description: '',
    category: 'food',
    coverMediaId: '',
    isFree: false,
  });
  assert.equal(created.category, 'food');
  const updated = await provider.updateStickerPackAsync('pack-1', { status: 'unlisted' });
  assert.equal(updated.status, 'unlisted');
  await provider.deleteStickerPackAsync('pack-1');
  const favorited = await provider.favoriteStickerPackAsync('pack-1');
  assert.equal(favorited.favoritedByMe, true);
  const unfavorited = await provider.unfavoriteStickerPackAsync('pack-1');
  assert.equal(unfavorited.favoritedByMe, false);
  const favorites = await provider.listFavoriteStickerPacksAsync();
  assert.equal(favorites.packs.length, 1);
  const stickers = await provider.listPackStickersAsync('pack-1');
  assert.equal(stickers.length, 1);
  assert.equal(stickers[0].packName, '山間小日子');
  assert.equal(stickers[0].mediaFileId, 'media-1');
  assert.equal(stickers[0].isAnimated, false);
  const added = await provider.addStickerAsync('pack-1', {
    mediaFileId: 'media-1',
    name: '新貼圖',
    isAnimated: true,
  });
  assert.equal(added.name, '新貼圖');
  assert.equal(added.isAnimated, true);
  await provider.removeStickerAsync('sticker-1');
  const results = await provider.searchStickersAsync('咖啡');
  assert.equal(results.stickers.length, 1);
  assert.equal(results.nextCursor, 'cursor-next');
  const recent = await provider.listRecentStickersAsync();
  assert.equal(recent.stickers.length, 1);
  await provider.recordStickerUseAsync('sticker-1');
  const addCall = calls.find(
    (item) => item.method === 'POST' && item.url.endsWith('/sticker-packs/pack-1/stickers'),
  );
  assert.deepEqual(JSON.parse(addCall.body), {
    media_file_id: 'media-1',
    name: '新貼圖',
    is_animated: true,
  });
});

test('provider listStickerPacksAsync clears the token on 401', async () => {
  const { provider, store } = await readyProvider(
    stickerRoutes({
      'GET /api/v1/sticker-packs': () => ({
        status: 401,
        body: { error: { code: 'ETHER_AUTH_UNAUTHENTICATED', message: '未認證。' } },
      }),
    }),
  );
  await assert.rejects(provider.listStickerPacksAsync(), (error) => error instanceof AuthError);
  assert.equal(store.getToken(), null);
  assert.equal(provider.getConnection().state, 'unauthorized');
});

test('demo provider sticker list hides paid-ungranted, unlisted and archived packs', async () => {
  const provider = createDemoProvider();
  const page = await provider.listStickerPacksAsync();
  assert.deepEqual(
    page.packs.map((pack) => pack.id),
    ['demo-pack-5', 'demo-pack-1', 'demo-pack-2', 'demo-pack-3'],
  );
  assert.equal(page.packs[3].isFree, false);
  const first = await provider.listStickerPacksAsync({ limit: 2 });
  assert.deepEqual(
    first.packs.map((pack) => pack.id),
    ['demo-pack-5', 'demo-pack-1'],
  );
  assert.notEqual(first.nextCursor, '');
  const second = await provider.listStickerPacksAsync({ cursor: first.nextCursor, limit: 2 });
  assert.deepEqual(
    second.packs.map((pack) => pack.id),
    ['demo-pack-2', 'demo-pack-3'],
  );
  await assert.rejects(provider.listStickerPacksAsync({ cursor: '!!!' }), /游標格式錯誤/);
  const animals = await provider.listStickerPacksAsync({ category: 'animals' });
  assert.deepEqual(
    animals.packs.map((pack) => pack.id),
    ['demo-pack-2'],
  );
  const mine = await provider.listStickerPacksAsync({ creatorId: 'me' });
  assert.deepEqual(
    mine.packs.map((pack) => pack.id),
    ['demo-pack-5'],
  );
  const byName = await provider.listStickerPacksAsync({ q: '貓' });
  assert.deepEqual(
    byName.packs.map((pack) => pack.id),
    ['demo-pack-2'],
  );
  await assert.rejects(provider.getStickerPackAsync('demo-pack-4'), /找不到貼圖包/);
  await assert.rejects(provider.getStickerPackAsync('demo-pack-6'), /找不到貼圖包/);
  await assert.rejects(provider.getStickerPackAsync('demo-pack-7'), /找不到貼圖包/);
  const granted = await provider.getStickerPackAsync('demo-pack-3');
  assert.equal(granted.isFree, false);
  const ownDraft = await provider.getStickerPackAsync('demo-pack-5');
  assert.equal(ownDraft.status, 'draft');
});

test('demo provider sticker favorites are idempotent and respect visibility', async () => {
  const provider = createDemoProvider();
  const initial = await provider.listFavoriteStickerPacksAsync();
  assert.deepEqual(
    initial.packs.map((pack) => pack.id),
    ['demo-pack-1'],
  );
  const favorited = await provider.favoriteStickerPackAsync('demo-pack-2');
  assert.equal(favorited.favoritedByMe, true);
  const again = await provider.favoriteStickerPackAsync('demo-pack-2');
  assert.equal(again.favoritedByMe, true);
  const listed = await provider.listFavoriteStickerPacksAsync();
  assert.deepEqual(
    listed.packs.map((pack) => pack.id),
    ['demo-pack-2', 'demo-pack-1'],
  );
  const unfavorited = await provider.unfavoriteStickerPackAsync('demo-pack-1');
  assert.equal(unfavorited.favoritedByMe, false);
  const againUnfavorite = await provider.unfavoriteStickerPackAsync('demo-pack-1');
  assert.equal(againUnfavorite.favoritedByMe, false);
  await assert.rejects(provider.favoriteStickerPackAsync('demo-pack-4'), /找不到貼圖包/);
  await assert.rejects(provider.favoriteStickerPackAsync('demo-pack-7'), /找不到貼圖包/);
});

test('demo provider sticker pack create, update and delete mirror the backend guards', async () => {
  const provider = createDemoProvider();
  await assert.rejects(
    provider.createStickerPackAsync({
      name: '  ',
      description: '',
      category: 'general',
      coverMediaId: '',
      isFree: true,
    }),
    /名稱不得為空/,
  );
  await assert.rejects(
    provider.createStickerPackAsync({
      name: 'x'.repeat(65),
      description: '',
      category: 'general',
      coverMediaId: '',
      isFree: true,
    }),
    /名稱過長/,
  );
  await assert.rejects(
    provider.createStickerPackAsync({
      name: '壞分類',
      description: '',
      category: 'bogus',
      coverMediaId: '',
      isFree: true,
    }),
    /分類/,
  );
  const created = await provider.createStickerPackAsync({
    name: '新包',
    description: '測試',
    category: 'animals',
    coverMediaId: '',
    isFree: true,
  });
  assert.equal(created.status, 'draft');
  assert.equal(created.stickerCount, 0);
  const found = await provider.getStickerPackAsync(created.id);
  assert.equal(found.name, '新包');
  const updated = await provider.updateStickerPackAsync(created.id, {
    name: '新包改名',
    status: 'published',
  });
  assert.equal(updated.name, '新包改名');
  assert.equal(updated.status, 'published');
  await assert.rejects(
    provider.updateStickerPackAsync('demo-pack-3', { name: '別人的包' }),
    /找不到貼圖包/,
  );
  await assert.rejects(provider.deleteStickerPackAsync('demo-pack-3'), /找不到貼圖包/);
  await provider.deleteStickerPackAsync(created.id);
  await assert.rejects(provider.getStickerPackAsync(created.id), /找不到貼圖包/);
  await assert.rejects(
    provider.updateStickerPackAsync(created.id, { name: '已封存' }),
    /找不到貼圖包/,
  );
});

test('demo provider pack stickers enforce the media gate and the 100-sticker cap', async () => {
  const provider = createDemoProvider();
  const stickers = await provider.listPackStickersAsync('demo-pack-1');
  assert.deepEqual(
    stickers.map((sticker) => sticker.id),
    ['demo-sticker-1', 'demo-sticker-2', 'demo-sticker-3'],
  );
  await assert.rejects(provider.listPackStickersAsync('demo-pack-4'), /找不到貼圖包/);
  await assert.rejects(
    provider.addStickerAsync('demo-pack-3', {
      mediaFileId: 'demo-file-photo',
      name: 'x',
      isAnimated: false,
    }),
    /找不到貼圖包/,
  );
  const ownPack = await provider.createStickerPackAsync({
    name: '上限測試包',
    description: '',
    category: 'general',
    coverMediaId: '',
    isFree: true,
  });
  await assert.rejects(
    provider.addStickerAsync(ownPack.id, {
      mediaFileId: 'no-such-media',
      name: 'x',
      isAnimated: false,
    }),
    /找不到要關聯的媒體檔案/,
  );
  await assert.rejects(
    provider.addStickerAsync(ownPack.id, {
      mediaFileId: 'demo-file-video',
      name: 'x',
      isAnimated: false,
    }),
    /尚未通過掃描/,
  );
  const added = await provider.addStickerAsync(ownPack.id, {
    mediaFileId: 'demo-file-photo',
    name: '新貼圖',
    isAnimated: false,
  });
  assert.equal(added.position, 0);
  const after = await provider.listPackStickersAsync(ownPack.id);
  assert.equal(after.length, 1);
  for (let index = 0; index < 99; index += 1) {
    await provider.addStickerAsync(ownPack.id, {
      mediaFileId: 'demo-file-photo',
      name: `填充 ${index}`,
      isAnimated: false,
    });
  }
  await assert.rejects(
    provider.addStickerAsync(ownPack.id, {
      mediaFileId: 'demo-file-photo',
      name: '第 101 張',
      isAnimated: false,
    }),
    /100 張/,
  );
  await assert.rejects(provider.removeStickerAsync('demo-sticker-4'), /找不到貼圖/);
  await provider.removeStickerAsync(added.id);
  const remaining = await provider.listPackStickersAsync(ownPack.id);
  assert.equal(
    remaining.some((sticker) => sticker.id === added.id),
    false,
  );
  await assert.rejects(provider.removeStickerAsync(added.id), /找不到貼圖/);
});

test('demo provider sticker search uses AND tokens over sticker and pack names', async () => {
  const provider = createDemoProvider();
  await assert.rejects(provider.searchStickersAsync('  '), /關鍵字不得為空/);
  const coffee = await provider.searchStickersAsync('咖啡');
  assert.deepEqual(
    coffee.stickers.map((sticker) => sticker.id),
    ['demo-sticker-1'],
  );
  const bothTokens = await provider.searchStickersAsync('咖啡 時間');
  assert.deepEqual(
    bothTokens.stickers.map((sticker) => sticker.id),
    ['demo-sticker-1'],
  );
  const packName = await provider.searchStickersAsync('山間');
  assert.deepEqual(
    packName.stickers.map((sticker) => sticker.id),
    ['demo-sticker-3', 'demo-sticker-2', 'demo-sticker-1'],
  );
  const filtered = await provider.searchStickersAsync('山間', { category: 'animals' });
  assert.equal(filtered.stickers.length, 0);
  const first = await provider.searchStickersAsync('山間', { limit: 1 });
  assert.equal(first.stickers.length, 1);
  assert.equal(first.stickers[0].id, 'demo-sticker-3');
  assert.notEqual(first.nextCursor, '');
  const second = await provider.searchStickersAsync('山間', {
    cursor: first.nextCursor,
    limit: 1,
  });
  assert.equal(second.stickers.length, 1);
  assert.equal(second.stickers[0].id, 'demo-sticker-2');
  await assert.rejects(provider.searchStickersAsync('山間', { cursor: '!!!' }), /游標格式錯誤/);
});

test('demo provider recent stickers dedupe and keep newest first', async () => {
  const provider = createDemoProvider();
  const initial = await provider.listRecentStickersAsync();
  assert.deepEqual(
    initial.stickers.map((sticker) => sticker.id),
    ['demo-sticker-6', 'demo-sticker-1'],
  );
  await provider.recordStickerUseAsync('demo-sticker-3');
  const afterFirst = await provider.listRecentStickersAsync();
  assert.deepEqual(
    afterFirst.stickers.map((sticker) => sticker.id),
    ['demo-sticker-3', 'demo-sticker-6', 'demo-sticker-1'],
  );
  await provider.recordStickerUseAsync('demo-sticker-3');
  const afterRepeat = await provider.listRecentStickersAsync();
  assert.deepEqual(
    afterRepeat.stickers.map((sticker) => sticker.id),
    ['demo-sticker-3', 'demo-sticker-6', 'demo-sticker-1'],
  );
  await provider.recordStickerUseAsync('demo-sticker-1');
  const afterMove = await provider.listRecentStickersAsync();
  const movedIds = afterMove.stickers.map((sticker) => sticker.id);
  assert.equal(movedIds.length, 3);
  assert.deepEqual([...movedIds].sort(), ['demo-sticker-1', 'demo-sticker-3', 'demo-sticker-6']);
  assert.equal(movedIds[2], 'demo-sticker-6');
  await assert.rejects(provider.recordStickerUseAsync('demo-sticker-12'), /找不到貼圖/);
  await assert.rejects(provider.recordStickerUseAsync('demo-sticker-8'), /找不到貼圖/);
  await assert.rejects(provider.recordStickerUseAsync('no-such-sticker'), /找不到貼圖/);
});
