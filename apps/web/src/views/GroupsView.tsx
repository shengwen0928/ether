import { useCallback, useEffect, useRef, useState } from 'react';
import type { ReactElement } from 'react';
import type { ChatDataProvider } from '../data/provider';
import type {
  GroupAnnouncementItem,
  GroupInvitationItem,
  GroupItem,
  GroupMemberItem,
  GroupSettingsItem,
} from '../data/types';
import { Avatar } from '../components/Avatar';
import { Badge } from '../components/Badge';
import { Button } from '../components/Button';
import { Dialog } from '../components/Dialog';
import { EmptyState } from '../components/EmptyState';
import { Input, TextArea } from '../components/Input';
import { Select } from '../components/Select';
import { useToast } from '../components/Toast';

export interface GroupsViewProps {
  provider: ChatDataProvider;
  dataMode?: 'demo' | 'api';
}

type LoadState = 'loading' | 'ready' | 'error' | 'unsupported';

interface GroupDetail {
  group: GroupItem;
  members: GroupMemberItem[];
  announcements: GroupAnnouncementItem[];
  invitations: GroupInvitationItem[];
  settings: GroupSettingsItem;
}

interface PendingConfirm {
  title: string;
  description: string;
  confirmLabel: string;
  danger: boolean;
  /** 離開／解散等動作已自行回到清單，不再重載舊的群組內容。 */
  reloadDetail?: boolean;
  action: () => Promise<void>;
}

interface AnnouncementDraft {
  id?: string;
  content: string;
}

function roleLabel(role: string): string {
  if (role === 'owner') {
    return '擁有者';
  }
  if (role === 'admin') {
    return '管理員';
  }
  return '一般成員';
}

function memberName(member: GroupMemberItem): string {
  if (member.displayName !== '') {
    return member.displayName;
  }
  return member.userId.length > 8 ? `${member.userId.slice(0, 8)}…` : member.userId;
}

function shortId(userId: string): string {
  return userId.length > 8 ? `${userId.slice(0, 8)}…` : userId;
}

function formatDateTime(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString('zh-Hant');
}

/**
 * Groups view (M6 follow-up): group list, creation, members and
 * roles, invitations, announcements and settings. The same
 * component serves demo mode (local fixtures, labelled as such)
 * and API mode (backend data); the provider methods are optional,
 * so absence renders the unsupported state instead of an error.
 * Permission gating follows the backend matrix (STATUS.md §2i):
 * managers manage members/invitations/announcements/settings,
 * only the owner transfers ownership, disbands or changes roles,
 * and the owner cannot leave. The backend remains the final
 * authority and answers 404 when a rule is violated.
 */
export function GroupsView({ provider, dataMode = 'demo' }: GroupsViewProps): ReactElement {
  const { notify } = useToast();
  const [listState, setListState] = useState<LoadState>('loading');
  const [groups, setGroups] = useState<GroupItem[]>([]);
  const [myInvitations, setMyInvitations] = useState<GroupInvitationItem[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detailState, setDetailState] = useState<LoadState>('loading');
  const [detail, setDetail] = useState<GroupDetail | null>(null);
  const [acting, setActing] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [transferOpen, setTransferOpen] = useState(false);
  const [announceDraft, setAnnounceDraft] = useState<AnnouncementDraft | null>(null);
  const [pendingConfirm, setPendingConfirm] = useState<PendingConfirm | null>(null);
  const [inviteInput, setInviteInput] = useState('');
  const [addMemberInput, setAddMemberInput] = useState('');
  const [settingsDraft, setSettingsDraft] = useState('admin');
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const currentUserId = provider.getCurrentUserId?.() ?? null;

  const loadList = useCallback(async (): Promise<void> => {
    const listGroups = provider.listGroupsAsync;
    const listMine = provider.listMyGroupInvitationsAsync;
    if (listGroups === undefined || listMine === undefined) {
      if (mountedRef.current) {
        setListState('unsupported');
      }
      return;
    }
    if (mountedRef.current) {
      setListState('loading');
    }
    try {
      const [fetchedGroups, fetchedInvitations] = await Promise.all([listGroups(), listMine()]);
      if (!mountedRef.current) {
        return;
      }
      setGroups(fetchedGroups);
      setMyInvitations(fetchedInvitations);
      setListState('ready');
    } catch {
      if (mountedRef.current) {
        setListState('error');
      }
    }
  }, [provider]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  const loadDetail = useCallback(
    async (groupId: string): Promise<void> => {
      const listMembers = provider.listGroupMembersAsync;
      const listAnnouncements = provider.listGroupAnnouncementsAsync;
      const getSettings = provider.getGroupSettingsAsync;
      const getGroup = provider.getGroupAsync;
      if (
        listMembers === undefined ||
        listAnnouncements === undefined ||
        getSettings === undefined ||
        getGroup === undefined
      ) {
        if (mountedRef.current) {
          setDetailState('unsupported');
        }
        return;
      }
      if (mountedRef.current) {
        setDetailState('loading');
      }
      try {
        const [group, members, announcements, settings] = await Promise.all([
          getGroup(groupId),
          listMembers(groupId),
          listAnnouncements(groupId),
          getSettings(groupId),
        ]);
        const self = members.find((item) => item.userId === currentUserId);
        const isManager = self?.role === 'owner' || self?.role === 'admin';
        let invitations: GroupInvitationItem[] = [];
        if (isManager && provider.listGroupInvitationsAsync !== undefined) {
          invitations = await provider.listGroupInvitationsAsync(groupId);
        }
        if (!mountedRef.current) {
          return;
        }
        setDetail({ group, members, announcements, invitations, settings });
        setSettingsDraft(settings.invitePolicy);
        setDetailState('ready');
      } catch {
        if (mountedRef.current) {
          setDetailState('error');
        }
      }
    },
    [provider, currentUserId],
  );

  const selectGroup = (groupId: string): void => {
    setSelectedId(groupId);
    setInviteInput('');
    setAddMemberInput('');
    void loadDetail(groupId);
  };

  const backToList = (): void => {
    setSelectedId(null);
    setDetail(null);
    void loadList();
  };

  const runAction = async (
    label: string,
    action: () => Promise<void>,
    reloadDetail = true,
  ): Promise<void> => {
    setActing(true);
    try {
      await action();
      notify({ title: '已完成', message: label, tone: 'success' });
      await loadList();
      if (reloadDetail && selectedId !== null) {
        await loadDetail(selectedId);
      }
    } catch {
      notify({ title: '操作失敗', message: '請檢查連線與權限後再試一次。', tone: 'error' });
    } finally {
      if (mountedRef.current) {
        setActing(false);
      }
    }
  };

  const myRole =
    detail === null
      ? ''
      : (detail.members.find((item) => item.userId === currentUserId)?.role ?? '');
  const isManager = myRole === 'owner' || myRole === 'admin';
  const isOwner = myRole === 'owner';
  const canInvite = detail !== null && (isManager || detail.settings.invitePolicy === 'all');
  const activeAnnouncement = detail?.announcements.find((item) => item.isActive) ?? null;

  const confirmPending = async (): Promise<void> => {
    if (pendingConfirm === null) {
      return;
    }
    const target = pendingConfirm;
    setPendingConfirm(null);
    await runAction(target.confirmLabel, target.action, target.reloadDetail ?? true);
  };

  const sourceLabel =
    dataMode === 'api' ? '群組、成員、邀請與公告來自後端 API。' : '展示資料，尚未連接後端。';

  if (selectedId !== null) {
    return (
      <div className="et-page et-scroll">
        <div className="et-page__narrow">
          <Button type="button" variant="secondary" size="sm" onClick={backToList}>
            返回群組清單
          </Button>
          {detailState === 'loading' ? (
            <p className="et-page__description">載入群組內容中…</p>
          ) : detailState === 'error' ? (
            <div>
              <p className="et-page__description">載入失敗，請檢查連線後再試一次。</p>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={() => {
                  void loadDetail(selectedId);
                }}
              >
                重新載入
              </Button>
            </div>
          ) : detailState === 'unsupported' || detail === null ? (
            <p className="et-page__description">目前的資料模式不提供群組內容。</p>
          ) : (
            <>
              <header>
                <h2 className="et-page__title">{detail.group.name}</h2>
                <p className="et-page__description">
                  {detail.group.description !== '' ? detail.group.description : '沒有群組介紹。'} ·{' '}
                  {detail.members.length} 位成員 · 我的角色：{roleLabel(myRole)} · {sourceLabel}
                </p>
              </header>

              {isManager ? (
                <div className="et-card">
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    disabled={acting}
                    onClick={() => {
                      setEditOpen(true);
                    }}
                  >
                    編輯群組資料
                  </Button>{' '}
                  {isOwner ? (
                    <>
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        disabled={acting}
                        onClick={() => {
                          setTransferOpen(true);
                        }}
                      >
                        轉移所有權
                      </Button>{' '}
                      <Button
                        type="button"
                        variant="danger"
                        size="sm"
                        disabled={acting}
                        onClick={() => {
                          setPendingConfirm({
                            title: '解散群組？',
                            description: `確定要解散「${detail.group.name}」嗎？解散後所有成員都無法再開啟此群組，訊息紀錄保留。`,
                            confirmLabel: '已解散群組。',
                            danger: true,
                            reloadDetail: false,
                            action: async () => {
                              await provider.disbandGroupAsync?.(selectedId);
                              backToList();
                            },
                          });
                        }}
                      >
                        解散群組
                      </Button>
                    </>
                  ) : null}
                </div>
              ) : null}

              {!isOwner ? (
                <div className="et-card">
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    disabled={acting}
                    onClick={() => {
                      setPendingConfirm({
                        title: '離開群組？',
                        description: `確定要離開「${detail.group.name}」嗎？離開後須重新受邀才能回來。`,
                        confirmLabel: '已離開群組。',
                        danger: false,
                        reloadDetail: false,
                        action: async () => {
                          await provider.leaveGroupAsync?.(selectedId);
                          backToList();
                        },
                      });
                    }}
                  >
                    離開群組
                  </Button>
                </div>
              ) : (
                <p className="et-page__description">
                  擁有者須先轉移所有權或解散群組，不得直接離開。
                </p>
              )}

              <section className="et-card" aria-labelledby="et-group-members-heading">
                <h3 className="et-card__title" id="et-group-members-heading">
                  成員
                </h3>
                <ul className="et-friend-list">
                  {detail.members.map((member) => (
                    <li className="et-friend-row" key={member.userId}>
                      <Avatar name={memberName(member)} size="md" />
                      <div className="et-friend-row__main">
                        <span className="et-friend-row__name">
                          {memberName(member)}{' '}
                          {member.role === 'owner' ? (
                            <Badge tone="accent">擁有者</Badge>
                          ) : member.role === 'admin' ? (
                            <Badge tone="success">管理員</Badge>
                          ) : null}
                        </span>
                        <span className="et-friend-row__meta">
                          於 {formatDateTime(member.joinedAt)} 加入
                        </span>
                      </div>
                      {isOwner && member.userId !== currentUserId && member.role !== 'owner' ? (
                        <>
                          <Button
                            type="button"
                            variant="secondary"
                            size="sm"
                            disabled={acting}
                            onClick={() => {
                              void runAction('已變更成員角色。', async () => {
                                await provider.setGroupMemberRoleAsync?.(
                                  selectedId,
                                  member.userId,
                                  member.role === 'admin' ? 'member' : 'admin',
                                );
                              });
                            }}
                          >
                            {member.role === 'admin' ? '設為一般成員' : '設為管理員'}
                          </Button>
                          <Button
                            type="button"
                            variant="danger"
                            size="sm"
                            disabled={acting}
                            onClick={() => {
                              setPendingConfirm({
                                title: '移除成員？',
                                description: `確定要將「${memberName(member)}」移出群組嗎？`,
                                confirmLabel: '已移除成員。',
                                danger: true,
                                action: async () => {
                                  await provider.removeGroupMemberAsync?.(
                                    selectedId,
                                    member.userId,
                                  );
                                },
                              });
                            }}
                          >
                            移除
                          </Button>
                        </>
                      ) : isManager &&
                        !isOwner &&
                        member.userId !== currentUserId &&
                        member.role === 'member' ? (
                        <Button
                          type="button"
                          variant="danger"
                          size="sm"
                          disabled={acting}
                          onClick={() => {
                            setPendingConfirm({
                              title: '移除成員？',
                              description: `確定要將「${memberName(member)}」移出群組嗎？`,
                              confirmLabel: '已移除成員。',
                              danger: true,
                              action: async () => {
                                await provider.removeGroupMemberAsync?.(selectedId, member.userId);
                              },
                            });
                          }}
                        >
                          移除
                        </Button>
                      ) : null}
                    </li>
                  ))}
                </ul>
                {isManager ? (
                  <div className="et-form-grid">
                    <Input
                      label="直接加入成員"
                      hint="使用者識別碼，以逗號分隔；展示模式可填聯絡人名稱或帳號。"
                      value={addMemberInput}
                      onChange={(event) => {
                        setAddMemberInput(event.target.value);
                      }}
                    />
                  </div>
                ) : null}
                {isManager ? (
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    disabled={acting || addMemberInput.trim() === ''}
                    onClick={() => {
                      const ids = addMemberInput
                        .split(',')
                        .map((item) => item.trim())
                        .filter((item) => item !== '');
                      void runAction('已加入成員。', async () => {
                        await provider.addGroupMembersAsync?.(selectedId, ids);
                      });
                      setAddMemberInput('');
                    }}
                  >
                    加入成員
                  </Button>
                ) : null}
              </section>

              <section className="et-card" aria-labelledby="et-group-invite-heading">
                <h3 className="et-card__title" id="et-group-invite-heading">
                  邀請
                </h3>
                {canInvite ? (
                  <>
                    <div className="et-form-grid">
                      <Input
                        label="被邀請者識別碼"
                        hint="使用者識別碼；展示模式可填聯絡人名稱或帳號。"
                        value={inviteInput}
                        onChange={(event) => {
                          setInviteInput(event.target.value);
                        }}
                      />
                    </div>
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      disabled={acting || inviteInput.trim() === ''}
                      onClick={() => {
                        const inviteeId = inviteInput.trim();
                        void runAction('已送出群組邀請。', async () => {
                          await provider.createGroupInvitationAsync?.(selectedId, inviteeId);
                        });
                        setInviteInput('');
                      }}
                    >
                      送出邀請
                    </Button>
                  </>
                ) : (
                  <p className="et-page__description">
                    邀請政策為「僅管理員」，一般成員無法送出邀請。
                  </p>
                )}
                {isManager ? (
                  detail.invitations.length === 0 ? (
                    <p className="et-page__description">沒有待處理的群組邀請。</p>
                  ) : (
                    <ul className="et-friend-list">
                      {detail.invitations.map((invitation) => (
                        <li className="et-friend-row" key={invitation.id}>
                          <Avatar name={shortId(invitation.inviteeId)} size="md" />
                          <div className="et-friend-row__main">
                            <span className="et-friend-row__name">
                              邀請 {shortId(invitation.inviteeId)}
                            </span>
                            <span className="et-friend-row__meta">
                              由 {shortId(invitation.inviterId)} 送出 ·{' '}
                              {formatDateTime(invitation.createdAt)}
                            </span>
                          </div>
                          <Button
                            type="button"
                            variant="secondary"
                            size="sm"
                            disabled={acting}
                            onClick={() => {
                              void runAction('已取消群組邀請。', async () => {
                                await provider.cancelGroupInvitationAsync?.(invitation.id);
                              });
                            }}
                          >
                            取消邀請
                          </Button>
                        </li>
                      ))}
                    </ul>
                  )
                ) : null}
              </section>

              <section className="et-card" aria-labelledby="et-group-announcements-heading">
                <h3 className="et-card__title" id="et-group-announcements-heading">
                  公告
                </h3>
                {isManager ? (
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    disabled={acting}
                    onClick={() => {
                      setAnnounceDraft({ content: '' });
                    }}
                  >
                    發佈公告
                  </Button>
                ) : null}
                {isManager && activeAnnouncement !== null ? (
                  <p className="et-page__description">發佈新公告會自動撤下目前的現行公告。</p>
                ) : null}
                {detail.announcements.length === 0 ? (
                  <p className="et-page__description">還沒有公告。</p>
                ) : (
                  <ul className="et-friend-list">
                    {detail.announcements.map((announcement) => (
                      <li className="et-friend-row" key={announcement.id}>
                        <div className="et-friend-row__main">
                          <span className="et-friend-row__name">
                            {announcement.isActive ? (
                              <Badge tone="success">現行公告</Badge>
                            ) : (
                              <Badge tone="warning">已撤下</Badge>
                            )}
                          </span>
                          <span className="et-friend-row__meta">{announcement.content}</span>
                          <span className="et-friend-row__meta">
                            由 {shortId(announcement.authorId)} 發佈 ·{' '}
                            {formatDateTime(announcement.publishedAt)}
                          </span>
                        </div>
                        {isManager && announcement.isActive ? (
                          <>
                            <Button
                              type="button"
                              variant="secondary"
                              size="sm"
                              disabled={acting}
                              onClick={() => {
                                setAnnounceDraft({
                                  id: announcement.id,
                                  content: announcement.content,
                                });
                              }}
                            >
                              編輯
                            </Button>
                            <Button
                              type="button"
                              variant="secondary"
                              size="sm"
                              disabled={acting}
                              onClick={() => {
                                setPendingConfirm({
                                  title: '撤下公告？',
                                  description:
                                    '確定要撤下這則現行公告嗎？撤下後須重新發佈才會再次顯示。',
                                  confirmLabel: '已撤下公告。',
                                  danger: false,
                                  action: async () => {
                                    await provider.retractGroupAnnouncementAsync?.(
                                      selectedId,
                                      announcement.id,
                                    );
                                  },
                                });
                              }}
                            >
                              撤下
                            </Button>
                          </>
                        ) : null}
                      </li>
                    ))}
                  </ul>
                )}
              </section>

              <section className="et-card" aria-labelledby="et-group-settings-heading">
                <h3 className="et-card__title" id="et-group-settings-heading">
                  群組設定
                </h3>
                {isManager ? (
                  <>
                    <div className="et-form-grid">
                      <Select
                        label="邀請政策"
                        aria-label="邀請政策"
                        value={settingsDraft}
                        options={[
                          { value: 'admin', label: '僅管理員可邀請' },
                          { value: 'all', label: '全體成員可邀請' },
                        ]}
                        onChange={(event) => {
                          setSettingsDraft(event.target.value);
                        }}
                      />
                    </div>
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      disabled={acting || settingsDraft === detail.settings.invitePolicy}
                      onClick={() => {
                        void runAction('已更新群組設定。', async () => {
                          await provider.updateGroupSettingsAsync?.(selectedId, settingsDraft);
                        });
                      }}
                    >
                      儲存設定
                    </Button>
                  </>
                ) : (
                  <p className="et-page__description">
                    邀請政策：
                    {detail.settings.invitePolicy === 'all' ? '全體成員可邀請' : '僅管理員可邀請'}
                    。僅管理員可變更設定。
                  </p>
                )}
              </section>
            </>
          )}
        </div>

        {editOpen && detail !== null ? (
          <EditGroupDialog
            group={detail.group}
            submitting={acting}
            onClose={() => {
              setEditOpen(false);
            }}
            onSubmit={async (name, description) => {
              await runAction('已更新群組資料。', async () => {
                await provider.updateGroupAsync?.(selectedId, { name, description });
              });
              setEditOpen(false);
            }}
          />
        ) : null}
        {transferOpen && detail !== null ? (
          <TransferDialog
            members={detail.members}
            currentUserId={currentUserId}
            submitting={acting}
            onClose={() => {
              setTransferOpen(false);
            }}
            onSubmit={async (newOwnerId) => {
              await runAction('已轉移群組所有權。', async () => {
                await provider.transferGroupOwnershipAsync?.(selectedId, newOwnerId);
              });
              setTransferOpen(false);
            }}
          />
        ) : null}
        {announceDraft !== null ? (
          <AnnouncementDialog
            draft={announceDraft}
            submitting={acting}
            onClose={() => {
              setAnnounceDraft(null);
            }}
            onSubmit={async (content) => {
              const announcementId = announceDraft.id;
              if (announcementId !== undefined) {
                await runAction('已更新公告。', async () => {
                  await provider.updateGroupAnnouncementAsync?.(
                    selectedId,
                    announcementId,
                    content,
                  );
                });
              } else {
                await runAction('已發佈公告。', async () => {
                  await provider.createGroupAnnouncementAsync?.(selectedId, content);
                });
              }
              setAnnounceDraft(null);
            }}
          />
        ) : null}
        {pendingConfirm !== null ? (
          <Dialog
            open={pendingConfirm !== null}
            onClose={() => {
              setPendingConfirm(null);
            }}
            title={pendingConfirm.title}
            description={pendingConfirm.description}
            footer={
              <>
                <Button
                  variant="secondary"
                  onClick={() => {
                    setPendingConfirm(null);
                  }}
                >
                  取消
                </Button>
                <Button
                  variant={pendingConfirm.danger ? 'danger' : 'primary'}
                  disabled={acting}
                  onClick={() => {
                    void confirmPending();
                  }}
                >
                  確認
                </Button>
              </>
            }
          >
            <p className="et-page__description">此操作會立即生效。</p>
          </Dialog>
        ) : null}
      </div>
    );
  }

  return (
    <div className="et-page et-scroll">
      <div className="et-page__narrow">
        <header>
          <h2 className="et-page__title">群組</h2>
          <p className="et-page__description">{sourceLabel}</p>
        </header>

        <section className="et-card" aria-labelledby="et-group-invitations-heading">
          <h3 className="et-card__title" id="et-group-invitations-heading">
            收到的群組邀請
          </h3>
          {listState === 'loading' ? (
            <p className="et-page__description">載入群組中…</p>
          ) : listState === 'error' ? (
            <div>
              <p className="et-page__description">載入失敗，請檢查連線後再試一次。</p>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={() => {
                  void loadList();
                }}
              >
                重新載入
              </Button>
            </div>
          ) : listState === 'unsupported' ? (
            <p className="et-page__description">目前的資料模式不提供群組清單。</p>
          ) : myInvitations.length === 0 ? (
            <p className="et-page__description">沒有待處理的群組邀請。</p>
          ) : (
            <ul className="et-friend-list">
              {myInvitations.map((invitation) => (
                <li className="et-friend-row" key={invitation.id}>
                  <Avatar name={invitation.groupName} size="md" />
                  <div className="et-friend-row__main">
                    <span className="et-friend-row__name">{invitation.groupName}</span>
                    <span className="et-friend-row__meta">
                      由 {shortId(invitation.inviterId)} 邀請 ·{' '}
                      {formatDateTime(invitation.createdAt)}
                    </span>
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    disabled={acting}
                    onClick={() => {
                      void runAction('已接受群組邀請。', async () => {
                        await provider.respondGroupInvitationAsync?.(invitation.id, true);
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
                      void runAction('已拒絕群組邀請。', async () => {
                        await provider.respondGroupInvitationAsync?.(invitation.id, false);
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

        <section className="et-card" aria-labelledby="et-groups-heading">
          <h3 className="et-card__title" id="et-groups-heading">
            我的群組
          </h3>
          {listState === 'loading' ? (
            <p className="et-page__description">載入群組中…</p>
          ) : listState === 'error' ? (
            <p className="et-page__description">載入失敗，請先重新載入上方的邀請區塊。</p>
          ) : listState === 'unsupported' ? (
            <p className="et-page__description">目前的資料模式不提供群組清單。</p>
          ) : groups.length === 0 ? (
            <EmptyState
              title="還沒有群組"
              description="建立第一個群組，邀請朋友一起加入。"
              action={
                <Button
                  onClick={() => {
                    setCreateOpen(true);
                  }}
                >
                  建立群組
                </Button>
              }
            />
          ) : (
            <>
              <ul className="et-friend-list">
                {groups.map((group) => (
                  <li className="et-friend-row" key={group.id}>
                    <Avatar name={group.name} size="md" />
                    <div className="et-friend-row__main">
                      <span className="et-friend-row__name">{group.name}</span>
                      <span className="et-friend-row__meta">
                        {group.memberCount} 位成員
                        {group.description !== '' ? ` · ${group.description}` : ''}
                      </span>
                    </div>
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => {
                        selectGroup(group.id);
                      }}
                    >
                      開啟
                    </Button>
                  </li>
                ))}
              </ul>
              <Button
                type="button"
                size="sm"
                onClick={() => {
                  setCreateOpen(true);
                }}
              >
                建立群組
              </Button>
            </>
          )}
        </section>
      </div>

      {createOpen ? (
        <CreateGroupDialog
          submitting={acting}
          onClose={() => {
            setCreateOpen(false);
          }}
          onSubmit={async (name, description, memberIds) => {
            await runAction('已建立群組。', async () => {
              const created = await provider.createGroupAsync?.({
                name,
                description,
                memberIds,
              });
              setCreateOpen(false);
              if (created !== undefined) {
                selectGroup(created.id);
              }
            });
          }}
        />
      ) : null}
      {pendingConfirm !== null ? (
        <Dialog
          open={pendingConfirm !== null}
          onClose={() => {
            setPendingConfirm(null);
          }}
          title={pendingConfirm.title}
          description={pendingConfirm.description}
          footer={
            <>
              <Button
                variant="secondary"
                onClick={() => {
                  setPendingConfirm(null);
                }}
              >
                取消
              </Button>
              <Button
                variant={pendingConfirm.danger ? 'danger' : 'primary'}
                disabled={acting}
                onClick={() => {
                  void confirmPending();
                }}
              >
                確認
              </Button>
            </>
          }
        >
          <p className="et-page__description">此操作會立即生效。</p>
        </Dialog>
      ) : null}
    </div>
  );
}

function CreateGroupDialog({
  submitting,
  onClose,
  onSubmit,
}: {
  submitting: boolean;
  onClose: () => void;
  onSubmit: (name: string, description: string, memberIds: string[]) => Promise<void>;
}): ReactElement {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [memberIds, setMemberIds] = useState('');
  const [error, setError] = useState('');

  return (
    <Dialog
      open
      onClose={onClose}
      title="建立群組"
      description="建立後你將成為群組擁有者。"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            取消
          </Button>
          <Button
            disabled={submitting || name.trim() === ''}
            onClick={() => {
              if (name.trim() === '') {
                setError('請輸入群組名稱。');
                return;
              }
              const ids = memberIds
                .split(',')
                .map((item) => item.trim())
                .filter((item) => item !== '');
              void onSubmit(name.trim(), description.trim(), ids);
            }}
          >
            建立
          </Button>
        </>
      }
    >
      <div className="et-form-grid">
        <Input
          label="群組名稱"
          value={name}
          error={error === '' ? undefined : error}
          onChange={(event) => {
            setName(event.target.value);
            setError('');
          }}
        />
        <TextArea
          label="群組介紹（選填）"
          value={description}
          onChange={(event) => {
            setDescription(event.target.value);
          }}
        />
        <Input
          label="初始成員（選填）"
          hint="使用者識別碼，以逗號分隔；展示模式可填聯絡人名稱或帳號。"
          value={memberIds}
          onChange={(event) => {
            setMemberIds(event.target.value);
          }}
        />
      </div>
    </Dialog>
  );
}

function EditGroupDialog({
  group,
  submitting,
  onClose,
  onSubmit,
}: {
  group: GroupItem;
  submitting: boolean;
  onClose: () => void;
  onSubmit: (name: string, description: string) => Promise<void>;
}): ReactElement {
  const [name, setName] = useState(group.name);
  const [description, setDescription] = useState(group.description);
  const [error, setError] = useState('');

  return (
    <Dialog
      open
      onClose={onClose}
      title="編輯群組資料"
      description="僅管理員可編輯群組資料。"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            取消
          </Button>
          <Button
            disabled={submitting || name.trim() === ''}
            onClick={() => {
              if (name.trim() === '') {
                setError('群組名稱不可為空。');
                return;
              }
              void onSubmit(name.trim(), description);
            }}
          >
            儲存
          </Button>
        </>
      }
    >
      <div className="et-form-grid">
        <Input
          label="群組名稱"
          value={name}
          error={error === '' ? undefined : error}
          onChange={(event) => {
            setName(event.target.value);
            setError('');
          }}
        />
        <TextArea
          label="群組介紹"
          value={description}
          onChange={(event) => {
            setDescription(event.target.value);
          }}
        />
      </div>
    </Dialog>
  );
}

function TransferDialog({
  members,
  currentUserId,
  submitting,
  onClose,
  onSubmit,
}: {
  members: GroupMemberItem[];
  currentUserId: string | null;
  submitting: boolean;
  onClose: () => void;
  onSubmit: (newOwnerId: string) => Promise<void>;
}): ReactElement {
  const candidates = members.filter(
    (item) => item.userId !== currentUserId && item.leftAt === null && item.role !== 'owner',
  );
  const first = candidates[0];
  const [target, setTarget] = useState(first?.userId ?? '');

  return (
    <Dialog
      open
      onClose={onClose}
      title="轉移所有權"
      description="轉移後你將成為管理員，新擁有者接任。僅擁有者可轉移。"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            取消
          </Button>
          <Button
            variant="danger"
            disabled={submitting || target === ''}
            onClick={() => {
              if (target !== '') {
                void onSubmit(target);
              }
            }}
          >
            轉移
          </Button>
        </>
      }
    >
      <div className="et-form-grid">
        <Select
          label="新擁有者"
          aria-label="新擁有者"
          value={target}
          options={candidates.map((item) => ({
            value: item.userId,
            label: `${memberName(item)}（${roleLabel(item.role)}）`,
          }))}
          onChange={(event) => {
            setTarget(event.target.value);
          }}
        />
      </div>
    </Dialog>
  );
}

function AnnouncementDialog({
  draft,
  submitting,
  onClose,
  onSubmit,
}: {
  draft: AnnouncementDraft;
  submitting: boolean;
  onClose: () => void;
  onSubmit: (content: string) => Promise<void>;
}): ReactElement {
  const [content, setContent] = useState(draft.content);
  const [error, setError] = useState('');

  return (
    <Dialog
      open
      onClose={onClose}
      title={draft.id !== undefined ? '編輯公告' : '發佈公告'}
      description={
        draft.id !== undefined
          ? '僅現行公告可編輯。'
          : '發佈後會自動撤下目前的現行公告。僅管理員可發佈。'
      }
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            取消
          </Button>
          <Button
            disabled={submitting || content.trim() === ''}
            onClick={() => {
              const trimmed = content.trim();
              if (trimmed === '' || [...trimmed].length > 1000) {
                setError('公告內容須為 1 至 1000 字。');
                return;
              }
              void onSubmit(trimmed);
            }}
          >
            {draft.id !== undefined ? '儲存' : '發佈'}
          </Button>
        </>
      }
    >
      <div className="et-form-grid">
        <TextArea
          label="公告內容"
          value={content}
          error={error === '' ? undefined : error}
          onChange={(event) => {
            setContent(event.target.value);
            setError('');
          }}
        />
      </div>
    </Dialog>
  );
}
