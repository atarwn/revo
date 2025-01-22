package ops

import (
	"encoding/binary"
	"evo/internal/crdt"
	"io"
	"os"

	"github.com/google/uuid"
)

// WriteOp writes a single CRDT op in binary
func WriteOp(w io.Writer, op crdt.Operation) error {
	// Format:
	// [1 byte opType]
	// [8 bytes lamport]
	// [16 bytes nodeID]
	// [16 bytes fileID]
	// [16 bytes lineID]
	// [4 bytes contentLen]
	// [content]
	buf := make([]byte, 1+8+16+16+16+4)
	buf[0] = byte(op.Type)
	binary.BigEndian.PutUint64(buf[1:9], op.Lamport)
	copy(buf[9:25], op.NodeID[:])
	copy(buf[25:41], op.FileID[:])
	copy(buf[41:57], op.LineID[:])

	contentBytes := []byte(op.Content)
	binary.BigEndian.PutUint32(buf[57:61], uint32(len(contentBytes)))
	if _, err := w.Write(buf); err != nil {
		return err
	}
	if len(contentBytes) > 0 {
		if _, err := w.Write(contentBytes); err != nil {
			return err
		}
	}
	return nil
}

func ReadOp(r io.Reader) (*crdt.Operation, error) {
	header := make([]byte, 1+8+16+16+16+4)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return nil, err
	}
	opType := crdt.OpType(header[0])
	lamport := binary.BigEndian.Uint64(header[1:9])
	var nodeID, fileID, lineID uuid.UUID
	copy(nodeID[:], header[9:25])
	copy(fileID[:], header[25:41])
	copy(lineID[:], header[41:57])
	contentLen := binary.BigEndian.Uint32(header[57:61])
	content := make([]byte, contentLen)
	if contentLen > 0 {
		if _, err := io.ReadFull(r, content); err != nil {
			return nil, err
		}
	}
	return &crdt.Operation{
		Type:    opType,
		Lamport: lamport,
		NodeID:  nodeID,
		FileID:  fileID,
		LineID:  lineID,
		Content: string(content),
	}, nil
}

func LoadAllOps(filename string) ([]crdt.Operation, error) {
	var out []crdt.Operation
	f, err := os.Open(filename)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	for {
		op, e := ReadOp(f)
		if e == io.EOF {
			break
		}
		if e != nil {
			// partial read => ignore or return
			return out, nil
		}
		out = append(out, *op)
	}
	return out, nil
}

func AppendOp(filename string, op crdt.Operation) error {
	if err := os.MkdirAll(dirOf(filename), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteOp(f, op)
}

func dirOf(fp string) string {
	for i := len(fp) - 1; i >= 0; i-- {
		if fp[i] == '/' || fp[i] == '\\' {
			return fp[:i]
		}
	}
	return "."
}
