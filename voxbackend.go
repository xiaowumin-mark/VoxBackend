package voxbackend

import (
	"context"

	"github.com/xiaowumin-mark/VoxBackend/player"
)

type Config = player.Config
type Player = player.Player
type State = player.State
type Event = player.Event
type EventType = player.EventType
type Callbacks = player.Callbacks
type Track = player.Track
type DSPMode = player.DSPMode

const (
	SeparatorFake = player.SeparatorFake
	SeparatorONNX = player.SeparatorONNX

	DSPModeOff  = player.DSPModeOff
	DSPModeOn   = player.DSPModeOn
	DSPModeAuto = player.DSPModeAuto
)

func DefaultConfig() Config {
	return player.DefaultConfig()
}

func NewPlayer(cfg Config) *Player {
	return player.New(cfg)
}

func Start(ctx context.Context, cfg Config) (*Player, error) {
	p := player.New(cfg)
	if err := p.Start(ctx); err != nil {
		return nil, err
	}
	return p, nil
}
