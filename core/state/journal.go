// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"math/big"

	"github.com/fulcrumchain/indigo/common"
	"github.com/fulcrumchain/indigo/log"
)

type journalEntry interface {
	undo(*StateDB)
}

type journal []journalEntry

type (
	// Changes to the account trie.
	createObjectChange struct {
		account *common.Address
	}
	resetObjectChange struct {
		prev *stateObject
	}
	suicideChange struct {
		account     *common.Address
		prev        bool // whether account had already suicided
		prevbalance *big.Int
	}

	// Changes to individual accounts.
	balanceChange struct {
		account *common.Address
		prev    *big.Int
	}
	nonceChange struct {
		account *common.Address
		prev    uint64
	}
	storageChange struct {
		account       *common.Address
		key, prevalue common.Hash
	}
	codeChange struct {
		account  *common.Address
		prevcode []byte
		prevhash common.Hash
	}

	// Changes to other state values.
	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txhash common.Hash
	}
	addPreimageChange struct {
		hash common.Hash
	}
	touchChange struct {
		account   *common.Address
		prev      bool
		prevDirty bool
	}
)

func (ch createObjectChange) undo(s *StateDB) {
	delete(s.stateObjects, *ch.account)
	delete(s.stateObjectsDirty, *ch.account)
}

func (ch resetObjectChange) undo(s *StateDB) {
	s.setStateObject(ch.prev)
}

func (ch suicideChange) undo(s *StateDB) {
	obj, err := s.getStateObject(*ch.account)
	if err != nil {
		log.Error("Failed to get state object", "err", err)
	}
	if obj != nil {
		obj.suicided = ch.prev
		obj.setBalance(ch.prevbalance)
	}
}

var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) undo(s *StateDB) {
	if !ch.prev && *ch.account != ripemd {
		so, err := s.getStateObject(*ch.account)
		if err != nil {
			log.Error("Failed to get state object", "err", err)
		}
		so.touched = ch.prev
		if !ch.prevDirty {
			delete(s.stateObjectsDirty, *ch.account)
		}
	}
}

func (ch balanceChange) undo(s *StateDB) {
	so, err := s.getStateObject(*ch.account)
	if err != nil {
		log.Error("Failed to get state object", "err", err)
	}
	so.setBalance(ch.prev)
}

func (ch nonceChange) undo(s *StateDB) {
	so, err := s.getStateObject(*ch.account)
	if err != nil {
		log.Error("Failed to get state object", "err", err)
	}
	so.setNonce(ch.prev)
}

func (ch codeChange) undo(s *StateDB) {
	so, err := s.getStateObject(*ch.account)
	if err != nil {
		log.Error("Failed to get state object", "err", err)
	}
	so.setCode(ch.prevhash, ch.prevcode)
}

func (ch storageChange) undo(s *StateDB) {
	so, err := s.getStateObject(*ch.account)
	if err != nil {
		log.Error("Failed to get state object", "err", err)
	}
	so.setState(ch.key, ch.prevalue)
}

func (ch refundChange) undo(s *StateDB) {
	s.refund = ch.prev
}

func (ch addLogChange) undo(s *StateDB) {
	logs := s.logs[ch.txhash]
	if len(logs) == 1 {
		delete(s.logs, ch.txhash)
	} else {
		s.logs[ch.txhash] = logs[:len(logs)-1]
	}
	s.logSize--
}

func (ch addPreimageChange) undo(s *StateDB) {
	delete(s.preimages, ch.hash)
}
