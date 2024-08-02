package main

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/yezzey-gp/yproxy/pkg/storage"

	"github.com/spf13/cobra"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/client"
	"github.com/yezzey-gp/yproxy/pkg/message"
	"github.com/yezzey-gp/yproxy/pkg/proc"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

var cfgPath string
var oldCfgPath string
var logLevel string
var decrypt bool
var encrypt bool
var offset uint64
var segmentPort int
var segmentNum int
var confirm bool

var rootCmd = &cobra.Command{
	Use:   "",
	Short: "",
}

var catCmd = &cobra.Command{
	Use:   "cat",
	Short: "cat",
	RunE: func(cmd *cobra.Command, args []string) error {

		err := config.LoadInstanceConfig(cfgPath)
		if err != nil {
			return err
		}

		instanceCnf := config.InstanceConfig()

		con, err := net.Dial("unix", instanceCnf.SocketPath)

		if err != nil {
			return err
		}

		defer con.Close()
		msg := message.NewCatMessage(args[0], decrypt, offset).Encode()
		_, err = con.Write(msg)
		if err != nil {
			return err
		}

		ylogger.Zero.Debug().Bytes("msg", msg).Msg("constructed cat message")

		_, err = io.Copy(os.Stdout, con)
		if err != nil {
			return err
		}

		return nil
	},
}

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "copy",
	RunE: func(cmd *cobra.Command, args []string) error {
		ylogger.Zero.Info().Msg("Execute copy command")
		err := config.LoadInstanceConfig(cfgPath)
		if err != nil {
			return err
		}
		instanceCnf := config.InstanceConfig()

		con, err := net.Dial("unix", instanceCnf.SocketPath)
		if err != nil {
			return err
		}

		defer con.Close()
		ylogger.Zero.Info().Str("name", args[0]).Msg("copy")
		msg := message.NewCopyMessage(args[0], oldCfgPath, encrypt, decrypt).Encode()
		_, err = con.Write(msg)
		if err != nil {
			return err
		}

		ylogger.Zero.Debug().Bytes("msg", msg).Msg("constructed copy msg")

		client := client.NewYClient(con)
		protoReader := proc.NewProtoReader(client)

		ansType, body, err := protoReader.ReadPacket()
		if err != nil {
			ylogger.Zero.Debug().Err(err).Msg("error while ans")
			return err
		}

		if ansType != message.MessageTypeReadyForQuery {
			return fmt.Errorf("failed to copy, msg: %v", body)
		}

		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete_garbage",
	Short: "delete_garbage",
	RunE: func(cmd *cobra.Command, args []string) error {
		ylogger.Zero.Info().Msg("Execute delete_garbage command")
		err := config.LoadInstanceConfig(cfgPath)
		if err != nil {
			return err
		}
		instanceCnf := config.InstanceConfig()

		con, err := net.Dial("unix", instanceCnf.SocketPath)
		if err != nil {
			return err
		}

		defer con.Close()
		ylogger.Zero.Info().Str("name", args[0]).Msg("delete")
		msg := message.NewDeleteMessage(args[0], segmentPort, segmentNum, confirm).Encode()
		_, err = con.Write(msg)
		if err != nil {
			return err
		}

		ylogger.Zero.Debug().Bytes("msg", msg).Msg("constructed delete msg")

		client := client.NewYClient(con)
		protoReader := proc.NewProtoReader(client)

		ansType, body, err := protoReader.ReadPacket()
		if err != nil {
			ylogger.Zero.Debug().Err(err).Msg("error while recieving answer")
			return err
		}

		if ansType != message.MessageTypeReadyForQuery {
			return fmt.Errorf("failed to delete, msg: %v", body)
		}

		return nil
	},
}

var putCmd = &cobra.Command{
	Use:   "put",
	Short: "put",
	RunE: func(cmd *cobra.Command, args []string) error {

		err := config.LoadInstanceConfig(cfgPath)
		if err != nil {
			return err
		}

		instanceCnf := config.InstanceConfig()

		con, err := net.Dial("unix", instanceCnf.SocketPath)

		if err != nil {
			return err
		}

		ycl := client.NewYClient(con)
		r := proc.NewProtoReader(ycl)

		defer con.Close()
		msg := message.NewPutMessage(args[0], encrypt).Encode()
		_, err = con.Write(msg)
		if err != nil {
			return err
		}

		ylogger.Zero.Debug().Bytes("msg", msg).Msg("constructed put message")

		const SZ = 65536
		chunk := make([]byte, SZ)
		for {
			n, err := os.Stdin.Read(chunk)
			if n > 0 {
				msg := message.NewCopyDataMessage()
				msg.Sz = uint64(n)
				msg.Data = make([]byte, msg.Sz)
				copy(msg.Data, chunk[:n])

				nwr, err := con.Write(msg.Encode())
				if err != nil {
					return err
				}

				ylogger.Zero.Debug().Int("len", nwr).Msg("written copy data msg")
			}

			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		ylogger.Zero.Debug().Msg("send command complete msg")

		msg = message.NewCommandCompleteMessage().Encode()
		_, err = con.Write(msg)
		if err != nil {
			return err
		}

		tp, _, err := r.ReadPacket()
		if err != nil {
			return err
		}

		if tp == message.MessageTypeReadyForQuery {
			// ok

			ylogger.Zero.Debug().Msg("got rfq")
		} else {
			return fmt.Errorf("failed to get rfq")
		}

		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list",
	RunE: func(cmd *cobra.Command, args []string) error {

		err := config.LoadInstanceConfig(cfgPath)
		if err != nil {
			return err
		}

		instanceCnf := config.InstanceConfig()

		con, err := net.Dial("unix", instanceCnf.SocketPath)

		if err != nil {
			return err
		}

		defer con.Close()
		msg := message.NewListMessage(args[0]).Encode()
		_, err = con.Write(msg)
		if err != nil {
			return err
		}

		ylogger.Zero.Debug().Bytes("msg", msg).Msg("constructed list message")

		ycl := client.NewYClient(con)
		r := proc.NewProtoReader(ycl)

		done := false
		res := make([]*storage.S3ObjectMeta, 0)
		for {
			if done {
				break
			}
			tp, body, err := r.ReadPacket()
			if err != nil {
				return err
			}

			switch tp {
			case message.MessageTypeObjectMeta:
				meta := message.ObjectMetaMessage{}
				meta.Decode(body)

				res = append(res, meta.Content...)
				break
			case message.MessageTypeReadyForQuery:
				done = true
				break
			default:
				return fmt.Errorf("Incorrect message type: %s", tp.String())
			}
		}

		for _, meta := range res {
			fmt.Printf("Object: {Name: \"%s\", size: %d}\n", meta.Path, meta.Size)
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "/etc/yproxy/yproxy.yaml", "path to yproxy config file")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "log level")

	catCmd.PersistentFlags().BoolVarP(&decrypt, "decrypt", "d", false, "decrypt external object or not")
	catCmd.PersistentFlags().Uint64VarP(&offset, "offset", "o", 0, "start offset for read")
	rootCmd.AddCommand(catCmd)

	copyCmd.PersistentFlags().BoolVarP(&decrypt, "decrypt", "d", false, "decrypt external object or not")
	copyCmd.PersistentFlags().BoolVarP(&encrypt, "encrypt", "e", false, "encrypt external object before put")
	copyCmd.PersistentFlags().StringVarP(&oldCfgPath, "old-config", "", "/etc/yproxy/yproxy.yaml", "path to old yproxy config file")
	rootCmd.AddCommand(copyCmd)

	putCmd.PersistentFlags().BoolVarP(&encrypt, "encrypt", "e", false, "encrypt external object before put")
	rootCmd.AddCommand(putCmd)

	rootCmd.AddCommand(listCmd)

	deleteCmd.PersistentFlags().IntVarP(&segmentPort, "port", "p", 6000, "port that segment is listening on")
	deleteCmd.PersistentFlags().IntVarP(&segmentNum, "segnum", "s", 0, "number of the segment")
	deleteCmd.PersistentFlags().BoolVarP(&confirm, "confirm", "", false, "confirm deletion")
	rootCmd.AddCommand(deleteCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		ylogger.Zero.Fatal().Err(err).Msg("")
	}
}
