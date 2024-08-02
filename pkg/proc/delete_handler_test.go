package proc_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/yezzey-gp/yproxy/pkg/message"
	mock "github.com/yezzey-gp/yproxy/pkg/mock"
	"github.com/yezzey-gp/yproxy/pkg/proc"
	"github.com/yezzey-gp/yproxy/pkg/storage"
)

func TestReworkingName(t *testing.T) {
	type TestCase struct {
		input    string
		expected string
	}

	testCases := []TestCase{
		{
			input:    "/segments_005/seg1/basebackups_005/yezzey/1663_16530_a4c5ad8305b83f07200b020694c36563_17660_1__DY_1_xlog_19649822496",
			expected: "1663_16530_a4c5ad8305b83f07200b020694c36563_17660_",
		},
		{
			input:    "1663_16530_a4c5ad8305b83f07200b020694c36563_17660_1__DY_1_xlog_19649822496",
			expected: "1663_16530_a4c5ad8305b83f07200b020694c36563_17660_",
		},
		{
			input:    "seg1/basebackups_005/yezzey/1663_16530_a4c5ad8305b83f07200b020694c36563_17660_1__DY_1_xlog_19649822496",
			expected: "1663_16530_a4c5ad8305b83f07200b020694c36563_17660_",
		},
		{
			input:    "1663_16530_a4c5ad8305b83f07200b020694c36563",
			expected: "1663_16530_a4c5ad8305b83f07200b020694c36563",
		},
		{
			input:    "1663___a4c5ad8305b83f07200b020694c36563___",
			expected: "1663___a4c5ad8305b83f07200b020694c36563_",
		},
		{
			input:    "file",
			expected: "file",
		},
	}

	for _, testCase := range testCases {
		ans := proc.ReworkFileName(testCase.input)
		assert.Equal(t, testCase.expected, ans)
	}
}

func TestFilesToDeletion(t *testing.T) {
	ctrl := gomock.NewController(t)

	msg := message.DeleteMessage{
		Name:    "path",
		Port:    6000,
		Segnum:  0,
		Confirm: false,
	}

	filesInStorage := []*storage.S3ObjectMeta{
		{Path: "1663_16530_not-deleted_18002_"},
		{Path: "1663_16530_deleted-after-backup_18002_"},
		{Path: "1663_16530_deleted-when-backup-start_18002_"},
		{Path: "1663_16530_deleted-before-backup_18002_"},
		{Path: "some_trash"},
	}
	storage := mock.NewMockStorageInteractor(ctrl)
	storage.EXPECT().ListPath(msg.Name).Return(filesInStorage, nil)

	backup := mock.NewMockBackupInterractor(ctrl)
	backup.EXPECT().GetFirstLSN(msg.Segnum).Return(uint64(1337), nil)

	vi := map[string]bool{
		"1663_16530_not-deleted_18002_":               true,
		"1663_16530_deleted-after-backup_18002_":      true,
		"1663_16530_deleted-when-backup-start_18002_": true,
	}
	ei := map[string]uint64{
		"1663_16530_deleted-after-backup_18002_":      uint64(1400),
		"1663_16530_deleted-when-backup-start_18002_": uint64(1337),
		"1663_16530_deleted-before-backup_18002_":     uint64(1300),
	}
	database := mock.NewMockDatabaseInterractor(ctrl)
	database.EXPECT().GetVirtualExpireIndexes(msg.Port).Return(vi, ei, nil)

	handler := proc.BasicDeleteHandler{
		StorageInterractor: storage,
		DbInterractor:      database,
		BackupInterractor:  backup,
	}

	list, err := handler.ListGarbageFiles(msg)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(list))
	assert.Equal(t, "1663_16530_deleted-before-backup_18002_", list[0])
	assert.Equal(t, "some_trash", list[1])
}
