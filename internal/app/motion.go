package app

import (
	"time"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
)

const (
	motionFast    = 120 * time.Millisecond
	motionNormal  = 160 * time.Millisecond
	motionEnter   = 200 * time.Millisecond
	motionStagger = 36 * time.Millisecond
)

var (
	easeEnter       = core.BezierEasing(core.EaseOut)
	easeExit        = core.BezierEasing(core.EaseIn)
	colorTransition = core.Transition{
		Duration:   motionFast,
		Easing:     core.EaseOut,
		Properties: core.TransitionBackground | core.TransitionBorder | core.TransitionOpacity,
	}
	interactiveTransition = core.Transition{
		Duration:   motionNormal,
		Easing:     core.EaseOut,
		Properties: core.TransitionBackground | core.TransitionBorder | core.TransitionTransform | core.TransitionOpacity | core.TransitionShadow,
	}
)

func scaleSteps(factor float32) []core.TransformStep {
	return []core.TransformStep{
		{Kind: core.TransformScaleX, Value: factor},
		{Kind: core.TransformScaleY, Value: factor},
	}
}

func translateYSteps(offset float32) []core.TransformStep {
	return []core.TransformStep{
		{Kind: core.TransformTranslateY, Value: offset},
	}
}

func rotateSteps(deg float32) []core.TransformStep {
	return []core.TransformStep{
		{Kind: core.TransformRotate, Value: deg},
	}
}

// buttonMotion adds smooth state transitions and subtle press scale while respecting reduced motion.
func buttonMotion(cx *core.Scope, pressScale float32) core.Style {
	if ui.ReduceMotion(cx) {
		return core.Style{Transition: colorTransition}
	}
	return core.Style{
		Transition: interactiveTransition,
		Active:     &core.Style{Transforms: scaleSteps(pressScale)},
	}
}

// rowCardMotion adds subtle hover lift and press feedback to clickable list cards.
func rowCardMotion(cx *core.Scope) core.Style {
	baseStyle := cardShadow(cx)
	if ui.ReduceMotion(cx) {
		baseStyle.Transition = colorTransition
		return baseStyle
	}
	baseStyle.Transition = interactiveTransition
	baseStyle.Hover = &core.Style{Transforms: translateYSteps(-1)}
	baseStyle.Active = &core.Style{Transforms: scaleSteps(0.992)}
	return baseStyle
}

// enterReveal fades and settles a newly mounted section or card without shifting layout bounds.
func enterReveal(cx *core.Scope, delay time.Duration, offsetY float64, child core.Node) core.Node {
	if ui.ReduceMotion(cx) {
		return child
	}
	opacity := core.NewAnimatedValue(cx, 0)
	y := core.NewAnimatedValue(cx, offsetY)
	cx.StartAnimation(core.Parallel(
		core.Timing(opacity, 1, core.TimingConfig{Duration: motionEnter, Delay: delay, Easing: easeEnter}),
		core.Timing(y, 0, core.TimingConfig{Duration: motionEnter, Delay: delay, Easing: easeEnter}),
	), nil)
	return core.View{
		Style: core.Style{
			Animated: &core.AnimatedStyle{
				Opacity:    opacity,
				TranslateY: y,
			},
		},
		Children: []core.Node{child},
	}
}

// enterRevealFlex is enterReveal for flex-1 containers (such as the chat workspace or settings body).
func enterRevealFlex(cx *core.Scope, offsetY float64, child core.Node) core.Node {
	if ui.ReduceMotion(cx) {
		return child
	}
	opacity := core.NewAnimatedValue(cx, 0)
	y := core.NewAnimatedValue(cx, offsetY)
	cx.StartAnimation(core.Parallel(
		core.Timing(opacity, 1, core.TimingConfig{Duration: motionEnter, Easing: easeEnter}),
		core.Timing(y, 0, core.TimingConfig{Duration: motionEnter, Easing: easeEnter}),
	), nil)
	return core.View{
		Class: "flex-1 min-w-0 min-h-0",
		Style: core.Style{
			Animated: &core.AnimatedStyle{
				Opacity:    opacity,
				TranslateY: y,
			},
		},
		Children: []core.Node{child},
	}
}

// popReveal smoothly pops in compact pills, badges, and floating callouts.
func popReveal(cx *core.Scope, offsetY float64, child core.Node) core.Node {
	if ui.ReduceMotion(cx) {
		return child
	}
	opacity := core.NewAnimatedValue(cx, 0)
	scale := core.NewAnimatedValue(cx, 0.94)
	y := core.NewAnimatedValue(cx, offsetY)
	cx.StartAnimation(core.Parallel(
		core.Timing(opacity, 1, core.TimingConfig{Duration: motionNormal, Easing: easeEnter}),
		core.Timing(scale, 1, core.TimingConfig{Duration: motionNormal, Easing: easeEnter}),
		core.Timing(y, 0, core.TimingConfig{Duration: motionNormal, Easing: easeEnter}),
	), nil)
	return core.View{
		Style: core.Style{
			Animated: &core.AnimatedStyle{
				Opacity:    opacity,
				Scale:      scale,
				TranslateY: y,
			},
		},
		Children: []core.Node{child},
	}
}

// presencePop mounts and unmounts compact transient feedback (such as the copy toast) with enter/exit motion.
func presencePop(when func() bool, then func(*core.Scope) core.Node) core.Node {
	return core.Presence{
		When: when,
		Then: func(cx *core.Scope) core.Node {
			content := then(cx)
			if ui.ReduceMotion(cx) {
				return content
			}
			opacity := core.NewAnimatedValue(cx, 0)
			scale := core.NewAnimatedValue(cx, 0.92)
			y := core.NewAnimatedValue(cx, 4)
			cx.StartAnimation(core.Parallel(
				core.Timing(opacity, 1, core.TimingConfig{Duration: motionNormal, Easing: easeEnter}),
				core.Timing(scale, 1, core.TimingConfig{Duration: motionNormal, Easing: easeEnter}),
				core.Timing(y, 0, core.TimingConfig{Duration: motionNormal, Easing: easeEnter}),
			), nil)
			cx.OnExit(func(done func()) {
				cx.StartAnimation(core.Parallel(
					core.Timing(opacity, 0, core.TimingConfig{Duration: 110 * time.Millisecond, Easing: easeExit}),
					core.Timing(scale, 0.94, core.TimingConfig{Duration: 110 * time.Millisecond, Easing: easeExit}),
					core.Timing(y, 2, core.TimingConfig{Duration: 110 * time.Millisecond, Easing: easeExit}),
				), func(bool) { done() })
			})
			return core.View{
				Style: core.Style{
					Animated: &core.AnimatedStyle{
						Opacity:    opacity,
						Scale:      scale,
						TranslateY: y,
					},
				},
				Children: []core.Node{content},
			}
		},
	}
}

// copyIconButton morphs its icon to an animated check mark in place after copying.
func copyIconButton(cx *core.Scope, name string, onCopy func()) core.Node {
	done := core.NewSignal(cx, false)
	var stop func() bool
	trigger := func() {
		onCopy()
		if stop != nil {
			stop()
		}
		done.Set(true)
		stop = cx.AfterFunc(1600*time.Millisecond, func() { done.Set(false) })
	}
	return base.Button{
		Label:   name,
		Tooltip: label(name, "text-xs"),
		Class:   "size-7 items-center justify-center rounded-md text-text-muted " + buttonStates,
		Style:   buttonMotion(cx, 0.90),
		OnPress: func(core.PressEvent) { trigger() },
		Children: []core.Node{
			core.Dynamic(func(cx *core.Scope) core.Node {
				if done.Get() {
					return popReveal(cx, 0, glyphSm(lucide.Check(), accent))
				}
				return glyphSm(lucide.Copy(), "")
			}),
		},
	}
}
