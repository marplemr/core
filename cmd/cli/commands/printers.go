package commands

import (
	"encoding/json"
	"fmt"
	"time"

	ds "github.com/c2h5oh/datasize"
	"github.com/docker/go-connections/nat"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sonm-io/core/insonmnia/node"
	pb "github.com/sonm-io/core/proto"
	"github.com/spf13/cobra"
)

func printTaskStatus(cmd *cobra.Command, id string, taskStatus *pb.TaskStatusReply) {
	if isSimpleFormat() {
		portsParsedOK := false
		ports := nat.PortMap{}
		if len(taskStatus.GetPorts()) > 0 {
			err := json.Unmarshal([]byte(taskStatus.GetPorts()), &ports)
			portsParsedOK = err == nil
		}

		cmd.Printf("Task %s (on %s):\r\n", id, taskStatus.MinerID)
		cmd.Printf("  Image:  %s\r\n", taskStatus.GetImageName())
		cmd.Printf("  Status: %s\r\n", taskStatus.GetStatus().String())
		cmd.Printf("  Uptime: %s\r\n", time.Duration(taskStatus.GetUptime()).String())

		if taskStatus.GetUsage() != nil {
			cmd.Println("  Resources:")
			cmd.Printf("    CPU: %d\r\n", taskStatus.Usage.GetCpu().GetTotal())
			cmd.Printf("    MEM: %s\r\n", ds.ByteSize(taskStatus.Usage.GetMemory().GetMaxUsage()).HR())
			if taskStatus.GetUsage().GetNetwork() != nil {
				cmd.Printf("    NET:\r\n")
				for i, net := range taskStatus.GetUsage().GetNetwork() {
					cmd.Printf("      %s:\r\n", i)
					cmd.Printf("        Tx/Rx bytes: %d/%d\r\n", net.TxBytes, net.RxBytes)
					cmd.Printf("        Tx/Rx packets: %d/%d\r\n", net.TxPackets, net.RxPackets)
					cmd.Printf("        Tx/Rx errors: %d/%d\r\n", net.TxErrors, net.RxErrors)
					cmd.Printf("        Tx/Rx dropped: %d/%d\r\n", net.TxDropped, net.RxDropped)
				}
			}
		}

		if portsParsedOK && len(ports) > 0 {
			cmd.Printf("  Ports:\r\n")
			for containerPort, host := range ports {
				if len(host) > 0 {
					cmd.Printf("    %s: %s:%s\r\n", containerPort, host[0].HostIP, host[0].HostPort)
				} else {
					cmd.Printf("    %s\r\n", containerPort)
				}
			}
		}
	} else {
		v := map[string]interface{}{
			"id":     id,
			"miner":  taskStatus.MinerID,
			"status": taskStatus.Status.String(),
			"image":  taskStatus.GetImageName(),
			"ports":  taskStatus.GetPorts(),
			"uptime": fmt.Sprintf("%d", time.Duration(taskStatus.GetUptime())),
		}
		if taskStatus.GetUsage() != nil {
			v["cpu"] = fmt.Sprintf("%d", taskStatus.GetUsage().GetCpu().GetTotal())
			v["mem"] = fmt.Sprintf("%d", taskStatus.GetUsage().GetMemory().GetMaxUsage())
			v["net"] = taskStatus.GetUsage().GetNetwork()
		}

		showJSON(cmd, v)
	}
}

func printNodeTaskStatus(cmd *cobra.Command, tasksMap map[string]*pb.TaskListReply_TaskInfo) {
	if isSimpleFormat() {
		for worker, tasks := range tasksMap {
			if len(tasks.GetTasks()) == 0 {
				cmd.Printf("Worker \"%s\" has no tasks\r\n", worker)
				continue
			}

			cmd.Printf("Worker \"%s\":\r\n", worker)
			i := 1
			for ID, status := range tasks.GetTasks() {
				up := time.Duration(status.GetUptime())
				cmd.Printf("  %d) %s \r\n     %s  %s (up: %v)\r\n",
					i, ID, status.Status.String(), status.ImageName, up.String())
				i++
			}
		}
	} else {
		showJSON(cmd, tasksMap)
	}
}

func printWorkerList(cmd *cobra.Command, lr *pb.ListReply) {
	if isSimpleFormat() {
		if len(lr.Info) == 0 {
			cmd.Printf("No workers connected\r\n")
			return
		}

		for addr, meta := range lr.Info {
			cmd.Printf("Worker: %s", addr)

			taskCount := len(meta.Values)
			if taskCount == 0 {
				cmd.Printf("\t\tIdle\r\n")
			} else {
				cmd.Printf("\t\t%d active task(s)\r\n", taskCount)
			}
		}
	} else {
		showJSON(cmd, lr)
	}
}

func printCpuInfo(cmd *cobra.Command, cap *pb.Capabilities) {
	for i, cpu := range cap.Cpu {
		cmd.Printf("    CPU%d: %d x %s\r\n", i, cpu.GetCores(), cpu.GetModelName())
	}
}

func printGpuInfo(cmd *cobra.Command, cap *pb.Capabilities) {
	if len(cap.Gpu) > 0 {
		for i, gpu := range cap.Gpu {
			cmd.Printf("    GPU%d: %s %s\r\n", i, gpu.VendorName, gpu.Name)
		}
	} else {
		cmd.Println("    GPU: None")
	}
}

func printMemInfo(cmd *cobra.Command, cap *pb.Capabilities) {
	cmd.Println("    RAM:")
	cmd.Printf("      Total: %s\r\n", ds.ByteSize(cap.Mem.GetTotal()).HR())
	cmd.Printf("      Used:  %s\r\n", ds.ByteSize(cap.Mem.GetUsed()).HR())
}

func printWorkerStatus(cmd *cobra.Command, workerID string, metrics *pb.InfoReply) {
	if isSimpleFormat() {
		cmd.Printf("Worker \"%s\":\r\n", workerID)

		if metrics.Capabilities != nil {
			cmd.Println("  Hardware:")
			printCpuInfo(cmd, metrics.Capabilities)
			printGpuInfo(cmd, metrics.Capabilities)
			printMemInfo(cmd, metrics.Capabilities)
		}

		if len(metrics.GetUsage()) == 0 {
			cmd.Println("  No active tasks")
		} else {
			cmd.Println("  Tasks:")
			i := 1
			for task := range metrics.Usage {
				cmd.Printf("    %d) %s\r\n", i, task)
				i++
			}
		}
	} else {
		showJSON(cmd, metrics)
	}
}

func printHubStatus(cmd *cobra.Command, stat *pb.HubStatusReply) {
	if isSimpleFormat() {
		cmd.Printf("Connected miners: %d\r\n", stat.MinerCount)
		cmd.Printf("Uptime:           %s\r\n", (time.Second * time.Duration(stat.Uptime)).String())
		cmd.Printf("Version:          %s %s\r\n", stat.Version, stat.Platform)
		cmd.Printf("Eth address:      %s\r\n", stat.EthAddr)
	} else {
		showJSON(cmd, stat)
	}
}

func printDeviceList(cmd *cobra.Command, devices *pb.DevicesReply) {
	if isSimpleFormat() {
		CPUs := devices.GetCPUs()
		GPUs := devices.GetGPUs()

		if len(CPUs) == 0 && len(GPUs) == 0 {
			cmd.Printf("No devices detected.\r\n")
			return
		}

		if len(CPUs) > 0 {
			cmd.Printf("CPUs:\r\n")
			for id, cpu := range CPUs {
				cmd.Printf(" %s: %s\r\n", id, cpu.Device.ModelName)
			}
		} else {
			cmd.Printf("No CPUs detected.\r\n")
		}

		if len(GPUs) > 0 {
			cmd.Printf("GPUs:\r\n")
			for id, gpu := range GPUs {
				cmd.Printf(" %s: %s\r\n", id, gpu.Device.Name)
			}
		} else {
			cmd.Printf("No GPUs detected.\r\n")
		}
	} else {
		showJSON(cmd, devices)
	}
}

func printDevicesProps(cmd *cobra.Command, props map[string]float64) {
	if isSimpleFormat() {
		for k, v := range props {
			cmd.Printf("%s = %f\r\n", k, v)
		}
	} else {
		showJSON(cmd, props)
	}
}

func printWorkerAclList(cmd *cobra.Command, list *pb.GetRegisteredWorkersReply) {
	if isSimpleFormat() {
		for i, id := range list.GetIds() {
			cmd.Printf("%d) %s\r\n", i+1, id.GetId())
		}

	} else {
		showJSON(cmd, list)
	}
}

func printTransactionInfo(cmd *cobra.Command, tx *types.Transaction) {
	if isSimpleFormat() {
		cmd.Printf("Hash:      %s\r\n", tx.Hash().String())
		cmd.Printf("Value:     %d\r\n", tx.Value().Uint64())
		cmd.Printf("To:        %s\r\n", tx.To().String())
		cmd.Printf("Cost:      %d\r\n", tx.Cost().Uint64())
		cmd.Printf("Gas:       %d\r\n", tx.Gas().Uint64())
		cmd.Printf("Gas price: %d\r\n", tx.GasPrice().Uint64())
	} else {
		showJSON(cmd, convertTransactionInfo(tx))
	}
}

func convertTransactionInfo(tx *types.Transaction) map[string]interface{} {
	return map[string]interface{}{
		"hash":      tx.Hash().String(),
		"value":     tx.Value().Uint64(),
		"to":        tx.To().String(),
		"cost":      tx.Cost().Uint64(),
		"gas":       tx.Gas().Uint64(),
		"gas_price": tx.GasPrice().Uint64(),
	}
}

func printSearchResults(cmd *cobra.Command, orders []*pb.Order) {
	if isSimpleFormat() {
		if len(orders) == 0 {
			cmd.Printf("No matching orders found")
			return
		}

		for i, order := range orders {
			cmd.Printf("%d) %s %s | price = %s\r\n", i+1, order.OrderType.String(), order.Id, order.Price)
		}
	} else {
		showJSON(cmd, map[string]interface{}{"orders": orders})
	}
}

func printOrderDetails(cmd *cobra.Command, order *pb.Order) {
	if isSimpleFormat() {
		cmd.Printf("ID:             %s\r\n", order.Id)
		cmd.Printf("Type:           %s\r\n", order.OrderType.String())
		cmd.Printf("Price:          %s\r\n", order.Price)

		cmd.Printf("SupplierID:     %s\r\n", order.SupplierID)
		cmd.Printf("SupplierRating: %d\r\n", order.Slot.SupplierRating)
		cmd.Printf("BuyerID:        %s\r\n", order.ByuerID)
		cmd.Printf("BuyerRating:    %d\r\n", order.Slot.BuyerRating)

		rs := order.Slot.Resources
		cmd.Printf("Resources:\r\n")
		cmd.Printf("  CPU:     %d\r\n", rs.CpuCores)
		cmd.Printf("  GPU:     %d\r\n", rs.GpuCount)
		cmd.Printf("  RAM:     %s\r\n", ds.ByteSize(rs.RamBytes).HR())
		cmd.Printf("  Storage: %s\r\n", ds.ByteSize(rs.Storage).HR())
		cmd.Printf("  Network: %s\r\n", rs.NetworkType.String())
		cmd.Printf("    In:   %s\r\n", ds.ByteSize(rs.NetTrafficIn).HR())
		cmd.Printf("    Out:  %s\r\n", ds.ByteSize(rs.NetTrafficOut).HR())
	} else {
		showJSON(cmd, order)
	}
}

func printProcessingOrders(cmd *cobra.Command, tasks *pb.GetProcessingReply) {
	if isSimpleFormat() {
		if len(tasks.GetOrders()) == 0 {
			cmd.Printf("No processing orders\r\n")
			return
		}

		for id, order := range tasks.GetOrders() {
			t := time.Unix(order.Timestamp.Seconds, 0)
			s := node.HandlerStatusString(uint8(order.Status))
			cmd.Printf("%s %s %s %s\r\n", t, id, s, order.Extra)
		}

	} else {
		showJSON(cmd, tasks)
	}
}

func printAskList(cmd *cobra.Command, slots *pb.SlotsReply) {
	if isSimpleFormat() {
		slots := slots.GetSlots()
		if len(slots) == 0 {
			cmd.Printf("No Ask Order configured\r\n")
			return
		}

		for id, slot := range slots {
			cmd.Printf(" ID:  %s", id)
			cmd.Printf(" CPU: %d Cores\r\n", slot.Resources.CpuCores)
			cmd.Printf(" GPU: %d Devices\r\n", slot.Resources.GpuCount)
			cmd.Printf(" RAM: %s\r\n", ds.ByteSize(slot.Resources.RamBytes).HR())
			cmd.Printf(" Net: %s\r\n", slot.Resources.NetworkType.String())
			cmd.Printf("     %s IN\r\n", ds.ByteSize(slot.Resources.NetTrafficIn).HR())
			cmd.Printf("     %s OUT\r\n", ds.ByteSize(slot.Resources.NetTrafficOut).HR())

			if slot.Geo != nil && slot.Geo.City != "" && slot.Geo.Country != "" {
				cmd.Printf(" Geo: %s, %s\r\n", slot.Geo.City, slot.Geo.Country)
			}
			cmd.Println("")
		}
	} else {
		showJSON(cmd, slots)
	}
}

func printVersion(cmd *cobra.Command, v string) {
	if isSimpleFormat() {
		cmd.Printf("Version: %s\r\n", v)
	} else {
		showJSON(cmd, map[string]string{"version": v})
	}

}

func printDealsList(cmd *cobra.Command, deals []*pb.Deal) {
	if isSimpleFormat() {
		if len(deals) == 0 {
			cmd.Println("No deals found")
			return
		}

		for _, deal := range deals {
			printDealInfo(cmd, deal)
			cmd.Println()
		}
	} else {
		showJSON(cmd, map[string]interface{}{"deals": deals})
	}

}

func printDealInfo(cmd *cobra.Command, deal *pb.Deal) {
	if isSimpleFormat() {
		start := time.Unix(deal.GetStartTime().GetSeconds(), int64(deal.GetStartTime().GetNanos()))
		end := time.Unix(deal.GetEndTime().GetSeconds(), int64(deal.GetEndTime().GetNanos()))

		cmd.Printf("ID:       %s\r\n", deal.GetId())
		cmd.Printf("Price:    %s\r\n", deal.GetPrice())
		cmd.Printf("Status:   %s\r\n", deal.GetStatus())
		cmd.Printf("Buyer:    %s\r\n", deal.GetBuyerID())
		cmd.Printf("Supplier: %s\r\n", deal.GetSupplierID())
		cmd.Printf("Start at: %s\r\n", start.Format(time.RFC3339))
		cmd.Printf("End at:   %s\r\n", end.Format(time.RFC3339))
	} else {
		showJSON(cmd, deal)
	}

}

func printID(cmd *cobra.Command, id string) {
	if isSimpleFormat() {
		cmd.Printf("ID = %s\r\n", id)
	} else {
		showJSON(cmd, map[string]string{"id": id})
	}
}

func printTaskStart(cmd *cobra.Command, start *pb.HubStartTaskReply) {
	if isSimpleFormat() {
		cmd.Printf("Task ID:      %s\r\n", start.Id)
		cmd.Printf("Hub Address:  %s\r\n", start.HubAddr)
		for _, end := range start.GetEndpoint() {
			cmd.Printf("  Endpoint:    %s\r\n", end)
		}
	} else {
		showJSON(cmd, start)
	}

}
