import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ReactElement } from 'react';
import { filterContacts, presenceLabel } from '../data/logic';
import type { ChatDataProvider } from '../data/provider';
import { useChatData } from '../data/provider';
import type {
  BlockItem,
  Contact,
  FriendItem,
  FriendRequestItem,
  SearchedUser,
} from '../data/types';
import { Avatar } from '../components/Avatar';
import { Badge } from '../components/Badge';
import { Button } from '../components/Button';
import { ContactItem } from '../components/ContactItem';
import { Dialog } from '../components/Dialog';
import { EmptyState } from '../components/EmptyState';
import { Input } from '../components/Input';
import { SearchField } from '../components/SearchField';
import { Select } from '../components/Select';
import { useToast } from '../components/Toast';

export interface ContactsViewProps {
  provider: ChatDataProvider;
  dataMode?: 'demo' | 'api';
  onStartConversation: (contactId: string) => void;
}

export function ContactsView({
  provider,
  dataMode = 'demo',
  onStartConversation,
}: ContactsViewProps): ReactElement {
  if (dataMode === 'api') {
    return <ApiContactsView provider={provider} />;
  }
  return <DemoContactsView provider={provider} onStartConversation={onStartConversation} />;
}

function DemoContactsView({
  provider,
  onStartConversation,
}: {
  provider: ChatDataProvider;
  onStartConversation: (contactId: string) => void;
}): ReactElement {
  const { notify } = useToast();
  const snapshot = useChatData(provider);
  const [query, setQuery] = useState('');
  const [group, setGroup] = useState('all');
  const [selected, setSelected] = useState<Contact | null>(null);

  const groups = useMemo(() => {
    const seen = new Set<string>();
    for (const contact of snapshot.contacts) {
      seen.add(contact.group);
    }
    return ['all', ...seen];
  }, [snapshot.contacts]);

  const visible = useMemo(
    () => filterContacts(snapshot.contacts, query, group),
    [snapshot.contacts, query, group],
  );

  const conversationFor = (contactId: string): string | undefined =>
    snapshot.conversations.find((conversation) => conversation.contactId === contactId)?.id;

  return (
    <div className="et-page et-scroll">
      <div className="et-page__narrow">
        <header>
          <h2 className="et-page__title">聯絡人</h2>
          <p className="et-page__description">
            {snapshot.contacts.length} 位聯絡人 · 展示資料，尚未連接後端。
          </p>
        </header>
        <div className="et-contacts__tools">
          <SearchField
            value={query}
            onChange={setQuery}
            placeholder="搜尋姓名、帳號或狀態"
            label="搜尋聯絡人"
          />
          <div style={{ minWidth: 160 }}>
            <Select
              label="分類"
              aria-label="依分類篩選聯絡人"
              value={group}
              options={groups.map((value) => ({
                value,
                label: value === 'all' ? '全部分類' : value,
              }))}
              onChange={(event) => {
                setGroup(event.target.value);
              }}
            />
          </div>
        </div>
        <div className="et-card" style={{ padding: 8 }}>
          {visible.length === 0 ? (
            <EmptyState
              title="沒有符合條件的聯絡人"
              description="試試其他關鍵字，或切換分類篩選。"
              action={
                <Button
                  variant="secondary"
                  onClick={() => {
                    setQuery('');
                    setGroup('all');
                  }}
                >
                  清除篩選
                </Button>
              }
            />
          ) : (
            <div className="et-contacts__list" role="list" aria-label="聯絡人列表">
              {visible.map((contact) => (
                <ContactItem
                  key={contact.id}
                  contact={contact}
                  onSelect={(contactId) => {
                    const found = snapshot.contacts.find((item) => item.id === contactId);
                    if (found !== undefined) {
                      setSelected(found);
                    }
                  }}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {selected !== null ? (
        <Dialog
          open={selected !== null}
          onClose={() => {
            setSelected(null);
          }}
          title={selected.name}
          description="聯絡人資訊（展示資料）"
          footer={
            <>
              <Button
                variant="secondary"
                onClick={() => {
                  setSelected(null);
                }}
              >
                關閉
              </Button>
              <Button
                onClick={() => {
                  const conversationId = conversationFor(selected.id);
                  if (conversationId === undefined) {
                    notify({
                      title: '展示模式',
                      message: '這位聯絡人還沒有對話，新對話建立尚未接線。',
                      tone: 'warning',
                    });
                    return;
                  }
                  setSelected(null);
                  onStartConversation(selected.id);
                }}
              >
                傳送訊息
              </Button>
            </>
          }
        >
          <div className="et-contact-detail">
            <Avatar name={selected.name} size="xl" presence={selected.presence} />
            <div className="et-contact-detail__row">
              <span className="et-contact-detail__key">帳號</span>
              <span className="et-contact-detail__value">{selected.handle}</span>
            </div>
            <div className="et-contact-detail__row">
              <span className="et-contact-detail__key">狀態</span>
              <span className="et-contact-detail__value">
                {selected.statusMessage !== ''
                  ? selected.statusMessage
                  : presenceLabel(selected.presence)}
              </span>
            </div>
            <div className="et-contact-detail__row">
              <span className="et-contact-detail__key">上次上線</span>
              <span className="et-contact-detail__value">{selected.lastSeen}</span>
            </div>
            <div className="et-contact-detail__row">
              <span className="et-contact-detail__key">分類</span>
              <span className="et-contact-detail__value">{selected.group}</span>
            </div>
          </div>
        </Dialog>
      ) : null}
    </div>
  );
}

type FriendsState = 'loading' | 'ready' | 'error' | 'unsupported';

interface PendingAction {
  kind: 'remove' | 'block';
  id: string;
  label: string;
}

/**
 * Backend-backed contacts view (M3, API mode): friends, pending
 * requests, public-id search and blocks all come from
 * /api/v1/friends/*. Starting a chat from here is not offered yet
 * (no create-conversation UI in this milestone).
 */
function ApiContactsView({ provider }: { provider: ChatDataProvider }): ReactElement {
  const { notify } = useToast();
  const [friendsState, setFriendsState] = useState<FriendsState>('loading');
  const [friends, setFriends] = useState<FriendItem[]>([]);
  const [incoming, setIncoming] = useState<FriendRequestItem[]>([]);
  const [outgoing, setOutgoing] = useState<FriendRequestItem[]>([]);
  const [blocks, setBlocks] = useState<BlockItem[]>([]);
  const [searchInput, setSearchInput] = useState('');
  const [searchMessage, setSearchMessage] = useState('');
  const [searching, setSearching] = useState(false);
  const [searchResult, setSearchResult] = useState<SearchedUser | null | undefined>(undefined);
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);
  const [acting, setActing] = useState(false);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const loadFriends = useCallback(async (): Promise<void> => {
    const listFriends = provider.listFriendsAsync;
    const listRequests = provider.listFriendRequestsAsync;
    const listBlocks = provider.listBlocksAsync;
    if (listFriends === undefined || listRequests === undefined || listBlocks === undefined) {
      if (mountedRef.current) {
        setFriendsState('unsupported');
      }
      return;
    }
    if (mountedRef.current) {
      setFriendsState('loading');
    }
    try {
      const [fetchedFriends, fetchedIncoming, fetchedOutgoing, fetchedBlocks] = await Promise.all([
        listFriends(true),
        listRequests('incoming'),
        listRequests('outgoing'),
        listBlocks(),
      ]);
      if (!mountedRef.current) {
        return;
      }
      setFriends(fetchedFriends);
      setIncoming(fetchedIncoming);
      setOutgoing(fetchedOutgoing);
      setBlocks(fetchedBlocks);
      setFriendsState('ready');
    } catch {
      if (mountedRef.current) {
        setFriendsState('error');
      }
    }
  }, [provider]);

  useEffect(() => {
    void loadFriends();
  }, [loadFriends]);

  const runAction = async (label: string, action: () => Promise<void>): Promise<void> => {
    setActing(true);
    try {
      await action();
      notify({ title: '已完成', message: label, tone: 'success' });
      await loadFriends();
    } catch {
      notify({ title: '操作失敗', message: '請檢查連線後再試一次。', tone: 'error' });
    } finally {
      if (mountedRef.current) {
        setActing(false);
      }
    }
  };

  const search = async (): Promise<void> => {
    const publicId = searchInput.trim();
    if (publicId === '' || provider.searchFriendAsync === undefined) {
      return;
    }
    setSearching(true);
    try {
      const found = await provider.searchFriendAsync(publicId);
      if (mountedRef.current) {
        setSearchResult(found);
      }
    } catch {
      if (mountedRef.current) {
        setSearchResult(undefined);
      }
      notify({ title: '搜尋失敗', message: '請檢查連線後再試一次。', tone: 'error' });
    } finally {
      if (mountedRef.current) {
        setSearching(false);
      }
    }
  };

  const sendRequest = async (): Promise<void> => {
    if (searchResult === undefined || searchResult === null) {
      return;
    }
    const target = searchResult;
    await runAction(`已送出好友邀請給「${target.displayName}」。`, async () => {
      await provider.sendFriendRequestAsync?.(target.id, searchMessage.trim());
    });
    setSearchResult(undefined);
    setSearchInput('');
    setSearchMessage('');
  };

  const confirmPendingAction = async (): Promise<void> => {
    if (pendingAction === null) {
      return;
    }
    const target = pendingAction;
    setPendingAction(null);
    if (target.kind === 'remove') {
      await runAction(`已解除與「${target.label}」的好友關係。`, async () => {
        await provider.removeFriendAsync?.(target.id);
      });
    } else {
      await runAction(`已封鎖「${target.label}」。`, async () => {
        await provider.blockUserAsync?.(target.id);
      });
    }
  };

  return (
    <div className="et-page et-scroll">
      <div className="et-page__narrow">
        <header>
          <h2 className="et-page__title">聯絡人</h2>
          <p className="et-page__description">{friends.length} 位好友 · 好友與邀請來自後端 API。</p>
        </header>

        <section className="et-card" aria-labelledby="et-friend-search-heading">
          <h3 className="et-card__title" id="et-friend-search-heading">
            以公開識別碼搜尋
          </h3>
          <div className="et-form-grid">
            <Input
              label="公開識別碼"
              value={searchInput}
              onChange={(event) => {
                setSearchInput(event.target.value);
              }}
            />
            <Input
              label="邀請附言（選填）"
              value={searchMessage}
              onChange={(event) => {
                setSearchMessage(event.target.value);
              }}
            />
          </div>
          <Button
            type="button"
            variant="secondary"
            disabled={searching || searchInput.trim() === ''}
            onClick={() => {
              void search();
            }}
          >
            {searching ? '搜尋中…' : '搜尋'}
          </Button>
          {searchResult === null ? (
            <p className="et-page__description">找不到這位使用者。</p>
          ) : searchResult !== undefined ? (
            <div className="et-friend-row">
              <Avatar name={searchResult.displayName} size="md" />
              <div className="et-friend-row__main">
                <span className="et-friend-row__name">{searchResult.displayName}</span>
                <span className="et-friend-row__meta">
                  @{searchResult.publicId}
                  {searchResult.statusMessage !== '' ? ` · ${searchResult.statusMessage}` : ''}
                </span>
              </div>
              <Button
                type="button"
                size="sm"
                disabled={acting}
                onClick={() => {
                  void sendRequest();
                }}
              >
                送出邀請
              </Button>
            </div>
          ) : null}
        </section>

        {friendsState === 'loading' ? (
          <p className="et-page__description">載入好友與邀請中…</p>
        ) : friendsState === 'error' ? (
          <div>
            <p className="et-page__description">載入失敗，請檢查連線後再試一次。</p>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={() => {
                void loadFriends();
              }}
            >
              重新載入
            </Button>
          </div>
        ) : friendsState === 'unsupported' ? (
          <p className="et-page__description">目前的資料模式不提供好友清單。</p>
        ) : (
          <>
            <section className="et-card" aria-labelledby="et-requests-in-heading">
              <h3 className="et-card__title" id="et-requests-in-heading">
                收到的邀請
              </h3>
              {incoming.length === 0 ? (
                <p className="et-page__description">沒有待處理的邀請。</p>
              ) : (
                <ul className="et-friend-list">
                  {incoming.map((request) => (
                    <li className="et-friend-row" key={request.id}>
                      <Avatar name={request.otherDisplayName} size="md" />
                      <div className="et-friend-row__main">
                        <span className="et-friend-row__name">{request.otherDisplayName}</span>
                        <span className="et-friend-row__meta">
                          @{request.otherPublicId}
                          {request.message !== '' ? ` · ${request.message}` : ''}
                        </span>
                      </div>
                      <Button
                        type="button"
                        size="sm"
                        disabled={acting}
                        onClick={() => {
                          void runAction('已接受好友邀請。', async () => {
                            await provider.respondFriendRequestAsync?.(request.id, true);
                          });
                        }}
                      >
                        接受
                      </Button>
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        disabled={acting}
                        onClick={() => {
                          void runAction('已拒絕好友邀請。', async () => {
                            await provider.respondFriendRequestAsync?.(request.id, false);
                          });
                        }}
                      >
                        拒絕
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <section className="et-card" aria-labelledby="et-requests-out-heading">
              <h3 className="et-card__title" id="et-requests-out-heading">
                送出的邀請
              </h3>
              {outgoing.length === 0 ? (
                <p className="et-page__description">沒有送出中的邀請。</p>
              ) : (
                <ul className="et-friend-list">
                  {outgoing.map((request) => (
                    <li className="et-friend-row" key={request.id}>
                      <Avatar name={request.otherDisplayName} size="md" />
                      <div className="et-friend-row__main">
                        <span className="et-friend-row__name">{request.otherDisplayName}</span>
                        <span className="et-friend-row__meta">@{request.otherPublicId}</span>
                      </div>
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        disabled={acting}
                        onClick={() => {
                          void runAction('已取消好友邀請。', async () => {
                            await provider.cancelFriendRequestAsync?.(request.id);
                          });
                        }}
                      >
                        取消邀請
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <section className="et-card" aria-labelledby="et-friends-heading">
              <h3 className="et-card__title" id="et-friends-heading">
                好友
              </h3>
              {friends.length === 0 ? (
                <p className="et-page__description">還沒有好友，先以公開識別碼搜尋並送出邀請。</p>
              ) : (
                <ul className="et-friend-list">
                  {friends.map((friend) => (
                    <li className="et-friend-row" key={friend.id}>
                      <Avatar name={friend.displayName} size="md" />
                      <div className="et-friend-row__main">
                        <span className="et-friend-row__name">
                          {friend.displayName}{' '}
                          {friend.isHidden ? <Badge tone="warning">已隱藏</Badge> : null}
                        </span>
                        <span className="et-friend-row__meta">
                          @{friend.publicId}
                          {friend.statusMessage !== '' ? ` · ${friend.statusMessage}` : ''}
                        </span>
                      </div>
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        disabled={acting}
                        onClick={() => {
                          void runAction(
                            friend.isHidden ? '已取消隱藏。' : '已隱藏此好友。',
                            async () => {
                              await provider.setFriendHiddenAsync?.(friend.id, !friend.isHidden);
                            },
                          );
                        }}
                      >
                        {friend.isHidden ? '取消隱藏' : '隱藏'}
                      </Button>
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        disabled={acting}
                        onClick={() => {
                          setPendingAction({
                            kind: 'remove',
                            id: friend.id,
                            label: friend.displayName,
                          });
                        }}
                      >
                        解除好友
                      </Button>
                      <Button
                        type="button"
                        variant="danger"
                        size="sm"
                        disabled={acting}
                        onClick={() => {
                          setPendingAction({
                            kind: 'block',
                            id: friend.id,
                            label: friend.displayName,
                          });
                        }}
                      >
                        封鎖
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <section className="et-card" aria-labelledby="et-blocks-heading">
              <h3 className="et-card__title" id="et-blocks-heading">
                已封鎖
              </h3>
              {blocks.length === 0 ? (
                <p className="et-page__description">沒有封鎖中的使用者。</p>
              ) : (
                <ul className="et-friend-list">
                  {blocks.map((block) => (
                    <li className="et-friend-row" key={block.userId}>
                      <Avatar name={block.displayName} size="md" />
                      <div className="et-friend-row__main">
                        <span className="et-friend-row__name">{block.displayName}</span>
                        <span className="et-friend-row__meta">
                          @{block.publicId}
                          {block.reason !== '' ? ` · ${block.reason}` : ''}
                        </span>
                      </div>
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        disabled={acting}
                        onClick={() => {
                          void runAction('已解除封鎖。', async () => {
                            await provider.unblockUserAsync?.(block.userId);
                          });
                        }}
                      >
                        解除封鎖
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </>
        )}
      </div>

      {pendingAction !== null ? (
        <Dialog
          open={pendingAction !== null}
          onClose={() => {
            setPendingAction(null);
          }}
          title={pendingAction.kind === 'remove' ? '解除好友？' : '封鎖使用者？'}
          description={
            pendingAction.kind === 'remove'
              ? `確定要解除與「${pendingAction.label}」的好友關係嗎？對方不會收到通知。`
              : `確定要封鎖「${pendingAction.label}」嗎？封鎖後對方無法再傳訊息給你，也不會知道自己被封鎖。`
          }
          footer={
            <>
              <Button
                variant="secondary"
                onClick={() => {
                  setPendingAction(null);
                }}
              >
                取消
              </Button>
              <Button
                variant={pendingAction.kind === 'block' ? 'danger' : 'primary'}
                disabled={acting}
                onClick={() => {
                  void confirmPendingAction();
                }}
              >
                {pendingAction.kind === 'remove' ? '解除好友' : '封鎖'}
              </Button>
            </>
          }
        >
          <p className="et-page__description">
            {pendingAction.kind === 'remove'
              ? '解除後如需恢復，須重新送出好友邀請。'
              : '解除封鎖後，好友關係不會自動恢復，須重新送出邀請。'}
          </p>
        </Dialog>
      ) : null}
    </div>
  );
}
