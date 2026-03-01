package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ai-gateway/internal/constants"
)

const classifierSwitchTimeoutMessage = "模型加载超时，请继续等待Ollama完成加载后重试"

func (h *RouterHandler) executeSwitchTask(taskID string) {
	if h.switchTaskStore == nil {
		return
	}
	task, err := h.switchTaskStore.Get(taskID)
	if err != nil || task == nil {
		return
	}

	now := h.nowFn()
	deadline := now.Add(constants.AdminClassifierSwitchAsyncMaxWait)
	task.Status = ClassifierSwitchTaskStatusRunning
	task.UpdatedAt = now.UnixMilli()
	task.DeadlineAt = deadline.UnixMilli()
	task.LastError = ""
	if err := h.switchTaskStore.Update(task); err != nil {
		return
	}

	for {
		now = h.nowFn()
		if !now.Before(deadline) {
			task.Status = ClassifierSwitchTaskStatusTimeout
			task.UpdatedAt = now.UnixMilli()
			task.LastError = classifierSwitchTimeoutMessage
			if err := h.switchTaskStore.Update(task); err != nil {
				return
			}
			return
		}

		task.Attempts++
		probeErr := h.probeSwitchFn(task.TargetModel, task.OriginalModel)
		if probeErr == nil {
			task.Status = ClassifierSwitchTaskStatusSuccess
			task.UpdatedAt = h.nowFn().UnixMilli()
			task.LastError = ""
			if err := h.switchTaskStore.Update(task); err != nil {
				return
			}
			return
		}

		task.UpdatedAt = h.nowFn().UnixMilli()
		task.LastError = strings.TrimSpace(probeErr.Error())
		task.Status = ClassifierSwitchTaskStatusRunning
		if err := h.switchTaskStore.Update(task); err != nil {
			return
		}

		h.sleepFn(constants.AdminClassifierSwitchProbeInterval)
	}
}

func (h *RouterHandler) probeAndApplyClassifierSwitch(targetModel, originalModel string) error {
	if h.router == nil {
		return errors.New("router is not initialized")
	}

	cfg := h.router.GetClassifierConfig()
	cfg.ActiveModel = targetModel
	h.router.SetClassifierConfig(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), constants.AdminClassifierSwitchProbeTimeout)
	defer cancel()
	health := h.router.GetClassifierHealth(ctx)
	if health == nil || !health.Healthy {
		cfg.ActiveModel = originalModel
		h.router.SetClassifierConfig(cfg)
		if health != nil && strings.TrimSpace(health.Message) != "" {
			return fmt.Errorf("%s", strings.TrimSpace(health.Message))
		}
		return errors.New("classifier health check failed")
	}

	h.loadConfig()
	h.mu.Lock()
	persistedConfig.Classifier = cfg
	h.mu.Unlock()
	if err := h.saveConfig(); err != nil {
		return fmt.Errorf("save switched classifier config failed: %w", err)
	}

	return nil
}
