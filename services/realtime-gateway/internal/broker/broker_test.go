package broker

import (
	"context"
	"testing"
	"time"
)

func TestMemoryBroker_PublishSubscribe(t *testing.T) {
	b := NewMemoryBroker()
	defer func() { _ = b.Close() }()
	ctx := context.Background()

	ch1, unsub1, err := b.Subscribe("conversation.c-1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer unsub1()
	ch2, unsub2, err := b.Subscribe("conversation.c-1")
	if err != nil {
		t.Fatalf("Subscribe 2: %v", err)
	}
	defer unsub2()
	other, unsubOther, err := b.Subscribe("conversation.c-2")
	if err != nil {
		t.Fatalf("Subscribe other: %v", err)
	}
	defer unsubOther()

	if err := b.Publish(ctx, Event{Topic: "conversation.c-1", Payload: []byte("hello")}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case ev := <-ch:
			if string(ev.Payload) != "hello" || ev.Topic != "conversation.c-1" {
				t.Fatalf("訂閱者 %d 收到 %+v", i, ev)
			}
		case <-time.After(time.Second):
			t.Fatalf("訂閱者 %d 未收到事件", i)
		}
	}
	select {
	case ev := <-other:
		t.Fatalf("其他主題不應收到事件：%+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestMemoryBroker_UnsubscribeStopsDelivery(t *testing.T) {
	b := NewMemoryBroker()
	defer func() { _ = b.Close() }()
	ch, unsub, err := b.Subscribe("conversation.c-1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	unsub()
	if err := b.Publish(context.Background(), Event{Topic: "conversation.c-1", Payload: []byte("x")}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if _, ok := <-ch; ok {
		t.Fatal("取消訂閱後通道應關閉")
	}
}

func TestMemoryBroker_DropsWhenFull(t *testing.T) {
	b := NewMemoryBroker()
	defer func() { _ = b.Close() }()
	ch, _, err := b.Subscribe("conversation.c-1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	// 不讀取：填滿 64 則緩衝後繼續發佈不得阻塞，且須計入丟棄。
	for i := 0; i < 200; i++ {
		if err := b.Publish(context.Background(), Event{Topic: "conversation.c-1", Payload: []byte("x")}); err != nil {
			t.Fatalf("Publish %d: %v", i, err)
		}
	}
	if b.Drops("conversation.c-1") == 0 {
		t.Fatal("緩衝滿時應計入丟棄")
	}
	// 通道仍可讀取（未被發佈者阻塞致死）。
	if _, ok := <-ch; !ok {
		t.Fatal("通道不應關閉")
	}
}

func TestJetStreamBroker_NotConfigured(t *testing.T) {
	b := NewJetStreamBroker()
	if err := b.Publish(context.Background(), Event{Topic: "t"}); err != ErrNotConfigured {
		t.Fatalf("Publish err = %v, want ErrNotConfigured", err)
	}
	if _, _, err := b.Subscribe("t"); err != ErrNotConfigured {
		t.Fatalf("Subscribe err = %v, want ErrNotConfigured", err)
	}
}
