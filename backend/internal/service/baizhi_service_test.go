package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSettingRepo struct {
	data map[string]string
}

func newMockSettingRepo() *mockSettingRepo {
	return &mockSettingRepo{data: make(map[string]string)}
}

func (m *mockSettingRepo) Get(ctx context.Context, key string) (*Setting, error) {
	val, ok := m.data[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: val}, nil
}

func (m *mockSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	val, ok := m.data[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return val, nil
}

func (m *mockSettingRepo) Set(ctx context.Context, key, value string) error {
	m.data[key] = value
	return nil
}

func (m *mockSettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	res := make(map[string]string)
	for _, k := range keys {
		if val, ok := m.data[k]; ok {
			res[k] = val
		}
	}
	return res, nil
}

func (m *mockSettingRepo) SetMultiple(ctx context.Context, settings map[string]string) error {
	for k, v := range settings {
		m.data[k] = v
	}
	return nil
}

func (m *mockSettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	return m.data, nil
}

func (m *mockSettingRepo) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func TestBaizhiService_DefaultsAndPersistence(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewBaizhiService(repo)
	ctx := context.Background()

	cfg, err := svc.GetConfig(ctx)
	require.NoError(t, err)
	assert.True(t, cfg.AuthRequired)
	assert.Equal(t, DefaultBaizhiScrapeBaseURL, cfg.WebScrape.BaseURL)
	assert.Equal(t, DefaultBaizhiSearchBaseURL, cfg.WebSearch.BaseURL)
	assert.Equal(t, DefaultBaizhiVoiceBaseURL, cfg.Voice.BaseURL)
	assert.Equal(t, DefaultBaizhiImageBaseURL, cfg.ImageGenerate.BaseURL)

	// Add keys to WebScrape
	cfg.WebScrape.Keys = append(cfg.WebScrape.Keys,
		BaizhiKeyEntry{
			ID:          "key-1",
			AccountName: "Account 1",
			APIKey:      "sk-test-key-1",
			Enabled:     true,
			Weight:      1,
		},
		BaizhiKeyEntry{
			ID:          "key-2",
			AccountName: "Account 2",
			APIKey:      "sk-test-key-2",
			Enabled:     false,
			Weight:      1,
		},
		BaizhiKeyEntry{
			ID:          "key-3",
			AccountName: "Account 3",
			APIKey:      "sk-test-key-3",
			Enabled:     true,
			Weight:      1,
		},
	)

	err = svc.UpdateConfig(ctx, cfg)
	require.NoError(t, err)

	// Round robin key selection: only key-1 and key-3 are enabled
	k1, _, _, err := svc.SelectKey(ctx, BaizhiServiceScrape)
	require.NoError(t, err)

	k2, _, _, err := svc.SelectKey(ctx, BaizhiServiceScrape)
	require.NoError(t, err)

	k3, _, _, err := svc.SelectKey(ctx, BaizhiServiceScrape)
	require.NoError(t, err)

	// Should alternate between key-1 and key-3
	assert.True(t, k1.ID == "key-1" || k1.ID == "key-3")
	assert.True(t, k2.ID == "key-1" || k2.ID == "key-3")
	assert.NotEqual(t, k1.ID, k2.ID)
	assert.Equal(t, k1.ID, k3.ID)

	// Service with no active keys
	_, _, _, err = svc.SelectKey(ctx, BaizhiServiceSearch)
	require.ErrorIs(t, err, ErrBaizhiNoActiveKeys)

	// Disabled service
	cfg.WebScrape.Enabled = false
	err = svc.UpdateConfig(ctx, cfg)
	require.NoError(t, err)

	_, _, _, err = svc.SelectKey(ctx, BaizhiServiceScrape)
	require.ErrorIs(t, err, ErrBaizhiServiceDisabled)
}
