package publication

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/blog"
	"github.com/google/uuid"
)

type Destination = blog.Destination
type Channel = blog.Channel
type DestinationEdit = blog.DestinationEdit
type DestinationUpdate = blog.DestinationUpdate
type AddedDestination = blog.AddedDestination
type Choice = blog.Choice

var PostEvents = blog.PostEvents
var ErrEventUnknown = blog.ErrEventUnknown
var ErrNotWebhook = blog.ErrNotWebhook

const KindWebhook = blog.KindWebhook
const KindDiscord = blog.KindDiscord
const DestinationUnverified = blog.DestinationUnverified
const DestinationActive = blog.DestinationActive
const DestinationDisabled = blog.DestinationDisabled

var ErrDestinationNotFound = blog.ErrDestinationNotFound
var ErrDestinationRefused = blog.ErrDestinationRefused
var ErrDestinationInactive = blog.ErrDestinationInactive
var ErrNotProven = blog.ErrNotProven

const EventVerification = blog.EventVerification

type ChannelEdit = blog.ChannelEdit

var ErrNotDiscord = blog.ErrNotDiscord

type DiscordRepair = blog.DiscordRepair
type DiscordRepairResult = blog.DiscordRepairResult
type RotatedSecret = blog.RotatedSecret

const SecretOverlap = blog.SecretOverlap

var DeliveryDelays = blog.DeliveryDelays
var DeliveryAttempts = blog.DeliveryAttempts

const SettledArrived = blog.SettledArrived
const SettledExhausted = blog.SettledExhausted
const SettledRefused = blog.SettledRefused
const SettledGone = blog.SettledGone
const SettledRemoved = blog.SettledRemoved
const SettledDisabled = blog.SettledDisabled
const SettledMoved = blog.SettledMoved
const SettledUnconfirmed = blog.SettledUnconfirmed

type Sender = blog.Sender
type Delivery = blog.Delivery
type DeliveryAttempt = blog.DeliveryAttempt

var ErrDeliveryNotFound = blog.ErrDeliveryNotFound
var ErrDeliveryUnsettled = blog.ErrDeliveryUnsettled
var ErrDeliveryUnsendable = blog.ErrDeliveryUnsendable

const DeliveryPending = blog.DeliveryPending
const DeliverySending = blog.DeliverySending
const DeliveryDelivered = blog.DeliveryDelivered
const DeliveryFailed = blog.DeliveryFailed
const DeliveryUnconfirmed = blog.DeliveryUnconfirmed
const AttemptDelivered = blog.AttemptDelivered
const AttemptRefused = blog.AttemptRefused
const AttemptUnreachable = blog.AttemptUnreachable
const AttemptUnconfirmed = blog.AttemptUnconfirmed
const DeliveryPoll = blog.DeliveryPoll

type Announcement = blog.Announcement
type DestinationPolicy = blog.DestinationPolicy

var ErrRoleRefused = blog.ErrRoleRefused

func (s *Service) PostDestinations(
	ctx context.Context,
	editor Editor,
	postID uuid.UUID,
) ([]Choice, error) {
	found, err := s.post(ctx, postID)
	if err != nil {
		return nil, err
	}
	if err := s.mayManage(ctx, editor, found); err != nil {
		return nil, err
	}
	allowed, err := s.PostChoices(ctx, found.GrantID)
	if err != nil {
		return nil, err
	}
	return blog.ActiveAmong(allowed), nil
}

func (s *Service) PostDeliveries(
	ctx context.Context,
	editor Editor,
	postID uuid.UUID,
) ([]Delivery, error) {
	post, err := s.post(ctx, postID)
	if err != nil {
		return nil, err
	}
	if err := s.mayManage(ctx, editor, post); err != nil {
		return nil, err
	}
	return s.Service.PostDeliveries(ctx, postID)
}

func (s *Service) SetGrantDestinations(ctx context.Context, actor, grantID uuid.UUID, in DestinationPolicy) error {
	current, err := s.grant(ctx, grantID)
	if err != nil {
		return err
	}
	return s.Service.SetGrantDestinations(ctx, actor, grantID, current.App.ID, current.Holder.ID, in)
}
