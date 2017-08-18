package commands

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sonm-io/core/cmd/cli/config"
	pb "github.com/sonm-io/core/proto"
	"github.com/stretchr/testify/assert"
)

func TestMinerStatusIdle(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().MinerStatus(gomock.Any(), gomock.Any()).AnyTimes().Return(&pb.InfoReply{}, nil)

	buf := initRootCmd(t, config.OutputModeSimple)
	minerStatusCmdRunner(rootCmd, "test", itr)
	out := buf.String()

	assert.Equal(t, "Miner: \"test\":\r\nMiner tasks:\n  No active tasks\n", out)
}

func TestMinerStatusData(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().
		MinerStatus(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(&pb.InfoReply{
			Stats: map[string]*pb.InfoReplyStats{
				"test": {
					CPU:    &pb.InfoReplyStatsCpu{TotalUsage: uint64(500)},
					Memory: &pb.InfoReplyStatsMemory{MaxUsage: uint64(2048)},
				},
			},
			Capabilities: &pb.Capabilities{
				Cpu: []*pb.CPUDevice{{Name: "i7", Vendor: "Intel", Mhz: 3000.0, Cores: 4}},
				Gpu: []*pb.GPUDevice{{Name: "GTX 1080Ti", Vendor: "NVidia"}},
				Mem: &pb.RAMDevice{Total: 1000000, Used: 500000},
			},
		}, nil)

	buf := initRootCmd(t, config.OutputModeSimple)
	minerStatusCmdRunner(rootCmd, "test", itr)
	out := buf.String()

	assert.Equal(t, "Miner: \"test\":\r\n  Hardware:\n    CPU0: 4 x i7\r\n    GPU0: NVidia GTX 1080Ti\r\n    RAM:\n      Total: 976.6 KB\r\n      Used:  488.3 KB\r\nMiner tasks:\n  ID: test\r\n      CPU: 500\r\n      RAM: 2.0 KB\r\n", out)
}

func TestMinerStatusJsonIdle(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().MinerStatus(gomock.Any(), gomock.Any()).AnyTimes().Return(&pb.InfoReply{}, nil)

	buf := initRootCmd(t, config.OutputModeJSON)
	minerStatusCmdRunner(rootCmd, "test", itr)
	out := buf.String()

	assert.Equal(t, "{}\n", out)
}

func TestMinerStatusJsonData(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().
		MinerStatus(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(&pb.InfoReply{
			Stats: map[string]*pb.InfoReplyStats{
				"test": {
					CPU:    &pb.InfoReplyStatsCpu{TotalUsage: uint64(500)},
					Memory: &pb.InfoReplyStatsMemory{MaxUsage: uint64(2048)},
				},
			},
		}, nil)

	buf := initRootCmd(t, config.OutputModeJSON)
	minerStatusCmdRunner(rootCmd, "test", itr)
	out := buf.String()

	info := &pb.InfoReply{}
	err := json.Unmarshal([]byte(out), &info)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(info.Stats))

	testStat, ok := info.Stats["test"]
	assert.True(t, ok)

	assert.Equal(t, uint64(2048), testStat.Memory.MaxUsage)
	assert.Equal(t, uint64(500), testStat.CPU.TotalUsage)
}

func TestMinerStatusFailed(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().MinerStatus(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, errors.New("error"))

	buf := initRootCmd(t, config.OutputModeSimple)
	minerStatusCmdRunner(rootCmd, "test", itr)
	out := buf.String()

	assert.Equal(t, "[ERR] Cannot get miner status: error\r\n", out)
}

func TestMinerStatusJsonFailed(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().MinerStatus(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, errors.New("some_error"))

	buf := initRootCmd(t, config.OutputModeJSON)
	minerStatusCmdRunner(rootCmd, "test", itr)
	out := buf.String()

	cmdErr, err := stringToCommandError(out)
	assert.NoError(t, err)

	assert.Equal(t, "some_error", cmdErr.Error)
	assert.Equal(t, "Cannot get miner status", cmdErr.Message)
}

func TestMinerListEmpty(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().MinerList(gomock.Any()).AnyTimes().Return(&pb.ListReply{}, nil)

	buf := initRootCmd(t, config.OutputModeSimple)
	minerListCmdRunner(rootCmd, itr)
	out := buf.String()

	assert.Equal(t, "No miners connected\r\n", out)
}

func TestMinerListData(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().
		MinerList(gomock.Any()).
		AnyTimes().
		Return(&pb.ListReply{
			Info: map[string]*pb.ListReply_ListValue{
				"test": {
					Values: []string{"task-1", "task-2"},
				},
			},
		}, nil)

	buf := initRootCmd(t, config.OutputModeSimple)
	minerListCmdRunner(rootCmd, itr)
	out := buf.String()

	assert.Equal(t, "Miner: test\r\nTasks:\n  1) task-1\r\n  2) task-2\r\n", out)
}

func TestMinerListDataNoTasks(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().
		MinerList(gomock.Any()).
		AnyTimes().
		Return(&pb.ListReply{
			Info: map[string]*pb.ListReply_ListValue{
				"test": {},
			},
		}, nil)

	buf := initRootCmd(t, config.OutputModeSimple)
	minerListCmdRunner(rootCmd, itr)
	out := buf.String()

	assert.Equal(t, "Miner: test\r\nMiner is idle\n", out)
}

func TestMinerListJsonEmpty(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().MinerList(gomock.Any()).AnyTimes().Return(&pb.ListReply{}, nil)

	buf := initRootCmd(t, config.OutputModeJSON)
	minerListCmdRunner(rootCmd, itr)
	out := buf.String()

	assert.Equal(t, "{}\n", out)
}

func TestMinerListJsonData(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().
		MinerList(gomock.Any()).
		AnyTimes().
		Return(&pb.ListReply{
			Info: map[string]*pb.ListReply_ListValue{
				"test": {
					Values: []string{"task-1", "task-2"},
				},
			},
		}, nil)

	buf := initRootCmd(t, config.OutputModeJSON)
	minerListCmdRunner(rootCmd, itr)
	out := buf.String()

	reply := &pb.ListReply{}
	err := json.Unmarshal([]byte(out), &reply)
	assert.NoError(t, err)

	assert.Len(t, reply.Info, 1)
	minerStat, ok := reply.Info["test"]
	assert.True(t, ok)

	assert.Len(t, minerStat.Values, 2)
	assert.Equal(t, "task-1", minerStat.Values[0])
	assert.Equal(t, "task-2", minerStat.Values[1])
}

func TestMinerListFailed(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().MinerList(gomock.Any()).AnyTimes().Return(nil, errors.New("some_error"))

	buf := initRootCmd(t, config.OutputModeSimple)
	minerListCmdRunner(rootCmd, itr)
	out := buf.String()
	assert.Equal(t, "[ERR] Cannot get miners list: some_error\r\n", out)
}

func TestMinerListJsonFailed(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().MinerList(gomock.Any()).AnyTimes().Return(nil, errors.New("some_error"))

	buf := initRootCmd(t, config.OutputModeJSON)
	minerListCmdRunner(rootCmd, itr)
	out := buf.String()

	cmdErr, err := stringToCommandError(out)
	assert.NoError(t, err)
	assert.Equal(t, "some_error", cmdErr.Error)
	assert.Equal(t, "Cannot get miners list", cmdErr.Message)
}

func TestMinerStatusMultiCPUAndGPU(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().
		MinerStatus(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(&pb.InfoReply{
			Stats: map[string]*pb.InfoReplyStats{
				"test": {
					CPU:    &pb.InfoReplyStatsCpu{TotalUsage: uint64(500)},
					Memory: &pb.InfoReplyStatsMemory{MaxUsage: uint64(2048)},
				},
			},
			Capabilities: &pb.Capabilities{
				Cpu: []*pb.CPUDevice{
					{Name: "Xeon E7-4850", Vendor: "Intel", Mhz: 2800.0, Cores: 14},
					{Name: "Xeon E7-8890", Vendor: "Intel", Mhz: 3400.0, Cores: 24},
				},
				Gpu: []*pb.GPUDevice{
					{Name: "GTX 1080Ti", Vendor: "NVidia"},
					{Name: "GTX 1080", Vendor: "NVidia"},
				},
				Mem: &pb.RAMDevice{Total: 1000000, Used: 500000},
			},
		}, nil)

	buf := initRootCmd(t, config.OutputModeSimple)
	minerStatusCmdRunner(rootCmd, "test", itr)
	out := buf.String()

	assert.Equal(t, "Miner: \"test\":\r\n  Hardware:\n    CPU0: 14 x Xeon E7-4850\r\n    CPU1: 24 x Xeon E7-8890\r\n    GPU0: NVidia GTX 1080Ti\r\n    GPU1: NVidia GTX 1080\r\n    RAM:\n      Total: 976.6 KB\r\n      Used:  488.3 KB\r\nMiner tasks:\n  ID: test\r\n      CPU: 500\r\n      RAM: 2.0 KB\r\n", out)
}

func TestMinerStatusNoGPU(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().
		MinerStatus(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(&pb.InfoReply{
			Stats: map[string]*pb.InfoReplyStats{
				"test": {
					CPU:    &pb.InfoReplyStatsCpu{TotalUsage: uint64(500)},
					Memory: &pb.InfoReplyStatsMemory{MaxUsage: uint64(2048)},
				},
			},
			Capabilities: &pb.Capabilities{
				Cpu: []*pb.CPUDevice{
					{Name: "Xeon E7-4850", Vendor: "Intel", Mhz: 2800.0, Cores: 14},
					{Name: "Xeon E7-8890", Vendor: "Intel", Mhz: 3400.0, Cores: 24},
				},
				Gpu: []*pb.GPUDevice{},
				Mem: &pb.RAMDevice{Total: 1000000, Used: 500000},
			},
		}, nil)

	buf := initRootCmd(t, config.OutputModeSimple)
	minerStatusCmdRunner(rootCmd, "test", itr)
	out := buf.String()

	assert.Equal(t, "Miner: \"test\":\r\n  Hardware:\n    CPU0: 14 x Xeon E7-4850\r\n    CPU1: 24 x Xeon E7-8890\r\n    GPU: None\n    RAM:\n      Total: 976.6 KB\r\n      Used:  488.3 KB\r\nMiner tasks:\n  ID: test\r\n      CPU: 500\r\n      RAM: 2.0 KB\r\n", out)
}

func TestMinerStatusWithName(t *testing.T) {
	itr := NewMockCliInteractor(gomock.NewController(t))
	itr.EXPECT().
		MinerStatus(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(&pb.InfoReply{
			Stats: map[string]*pb.InfoReplyStats{
				"test": {
					CPU:    &pb.InfoReplyStatsCpu{TotalUsage: uint64(500)},
					Memory: &pb.InfoReplyStatsMemory{MaxUsage: uint64(2048)},
				},
			},
			Name: "fb402dcf-ff56-465e-8aad-bcef7ca1ef9a",
		}, nil)

	buf := initRootCmd(t, config.OutputModeSimple)
	minerStatusCmdRunner(rootCmd, "test", itr)
	out := buf.String()

	assert.Equal(t, "Miner: \"test\" (fb402dcf-ff56-465e-8aad-bcef7ca1ef9a):\r\nMiner tasks:\n  ID: test\r\n      CPU: 500\r\n      RAM: 2.0 KB\r\n", out)
}
