import { useEffect, useLayoutEffect, useMemo, useState } from "react";

const spotlightPadding = 8;

function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max);
}

function getSpotlightRect(rect) {
  if (!rect) return null;
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;
  const top = clamp(rect.top - spotlightPadding, 0, viewportHeight);
  const left = clamp(rect.left - spotlightPadding, 0, viewportWidth);
  const right = clamp(rect.right + spotlightPadding, 0, viewportWidth);
  const bottom = clamp(rect.bottom + spotlightPadding, 0, viewportHeight);

  return {
    top,
    left,
    right,
    bottom,
    width: Math.max(0, right - left),
    height: Math.max(0, bottom - top),
  };
}

function getPopoverPosition(targetRect, popoverSize) {
  const margin = 14;
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;
  const spaceBelow = viewportHeight - targetRect.bottom;
  const spaceAbove = targetRect.top;
  const spaceRight = viewportWidth - targetRect.right;
  const spaceLeft = targetRect.left;
  let placement = "bottom";

  if (spaceBelow < popoverSize.height + margin && spaceAbove > spaceBelow) placement = "top";
  if (Math.max(spaceBelow, spaceAbove) < popoverSize.height + margin) {
    placement = spaceRight > spaceLeft ? "right" : "left";
  }

  if (placement === "top") {
    return {
      placement,
      top: clamp(targetRect.top - popoverSize.height - margin, margin, viewportHeight - popoverSize.height - margin),
      left: clamp(targetRect.left, margin, viewportWidth - popoverSize.width - margin),
    };
  }

  if (placement === "right") {
    return {
      placement,
      top: clamp(targetRect.top, margin, viewportHeight - popoverSize.height - margin),
      left: clamp(targetRect.right + margin, margin, viewportWidth - popoverSize.width - margin),
    };
  }

  if (placement === "left") {
    return {
      placement,
      top: clamp(targetRect.top, margin, viewportHeight - popoverSize.height - margin),
      left: clamp(targetRect.left - popoverSize.width - margin, margin, viewportWidth - popoverSize.width - margin),
    };
  }

  return {
    placement,
    top: clamp(targetRect.bottom + margin, margin, viewportHeight - popoverSize.height - margin),
    left: clamp(targetRect.left, margin, viewportWidth - popoverSize.width - margin),
  };
}

export function TourOverlay({ children }) {
  return <div className="tour-overlay">{children}</div>;
}

export function TourSpotlight({ rect }) {
  const spotlight = getSpotlightRect(rect);
  if (!spotlight) return null;
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;

  return (
    <>
      <div className="tour-scrim" style={{ top: 0, left: 0, width: "100vw", height: spotlight.top }} />
      <div className="tour-scrim" style={{ top: spotlight.bottom, left: 0, width: "100vw", height: Math.max(0, viewportHeight - spotlight.bottom) }} />
      <div className="tour-scrim" style={{ top: spotlight.top, left: 0, width: spotlight.left, height: spotlight.height }} />
      <div className="tour-scrim" style={{ top: spotlight.top, left: spotlight.right, width: Math.max(0, viewportWidth - spotlight.right), height: spotlight.height }} />
      <div
        className="tour-spotlight"
        style={{
          top: spotlight.top,
          left: spotlight.left,
          width: spotlight.width,
          height: spotlight.height,
        }}
      />
    </>
  );
}

export function TourPopover({
  step,
  index,
  total,
  position,
  onBack,
  onNext,
  onSkip,
  backLabel = "Back",
  nextLabel = "Next",
  doneLabel = "Selesai",
  skipLabel = "Skip",
  displayIndex,
  displayTotal,
}) {
  if (!step || !position) return null;
  const visibleIndex = displayIndex ?? index;
  const visibleTotal = displayTotal ?? total;
  const isFirst = visibleIndex <= 0;
  const isLast = index === total - 1;

  return (
    <div
      className={`tour-popover ${position.placement}`}
      style={{ top: position.top, left: position.left }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="tour-step-title"
    >
      <div className="tour-step-label">Step {visibleIndex + 1} dari {visibleTotal}</div>
      <h3 id="tour-step-title">{step.title}</h3>
      <p>{step.text}</p>
      <div className="tour-actions">
        <button className="btn btn-secondary" type="button" onClick={onBack} disabled={isFirst}>{backLabel}</button>
        <button className="btn btn-ghost" type="button" onClick={onSkip}>{skipLabel}</button>
        <button className="btn btn-primary" type="button" onClick={onNext}>{isLast ? doneLabel : nextLabel}</button>
      </div>
    </div>
  );
}

export function GuidedTour({
  open,
  steps,
  storageKey,
  onClose,
  onStepChange,
  onFinish,
  onSkipTour,
  backLabel,
  nextLabel,
  doneLabel,
  skipLabel,
  displayIndex,
  displayTotal,
  onBackTour,
}) {
  const [index, setIndex] = useState(0);
  const [targetRect, setTargetRect] = useState(null);
  const [position, setPosition] = useState(null);
  const step = steps[index];
  const popoverSize = useMemo(() => {
    const viewportWidth = typeof window === "undefined" ? 390 : window.innerWidth;
    return { width: Math.min(360, viewportWidth - 32), height: 210 };
  }, []);

  useEffect(() => {
    if (!open) return;
    setIndex(0);
  }, [open]);

  useEffect(() => {
    if (!open) return undefined;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [open]);

  useEffect(() => {
    if (!open || !step) return;
    onStepChange?.(step, index);
  }, [index, onStepChange, open, step]);

  useLayoutEffect(() => {
    if (!open || !step) return undefined;
    let cancelled = false;
    let frame = 0;

    function measure() {
      const target = document.querySelector(`[data-tour="${step.target}"]`);
      if (!target) {
        setTargetRect(null);
        setPosition(null);
        if (!cancelled) {
          frame = window.requestAnimationFrame(() => {
            if (index < steps.length - 1) {
              setIndex((current) => Math.min(current + 1, steps.length - 1));
            } else {
              if (storageKey) window.localStorage.setItem(storageKey, "skipped");
              onClose?.();
            }
          });
        }
        return;
      }
      target.scrollIntoView({ block: "center", inline: "center", behavior: "smooth" });
      frame = window.requestAnimationFrame(() => {
        const rect = target.getBoundingClientRect();
        if (cancelled) return;
        setTargetRect(rect);
        setPosition(getPopoverPosition(rect, popoverSize));
      });
    }

    const timer = window.setTimeout(measure, 220);
    window.addEventListener("resize", measure);
    window.addEventListener("scroll", measure, true);
    window.visualViewport?.addEventListener("resize", measure);
    window.visualViewport?.addEventListener("scroll", measure);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
      window.cancelAnimationFrame(frame);
      window.removeEventListener("resize", measure);
      window.removeEventListener("scroll", measure, true);
      window.visualViewport?.removeEventListener("resize", measure);
      window.visualViewport?.removeEventListener("scroll", measure);
    };
  }, [index, onClose, open, popoverSize, step, steps.length, storageKey]);

  if (!open || !step) return null;

  function closeWithStatus(status) {
    if (storageKey) window.localStorage.setItem(storageKey, status);
    if (status === "completed") onFinish?.();
    if (status === "skipped") onSkipTour?.();
    onClose?.();
  }

  function next() {
    if (index >= steps.length - 1) {
      closeWithStatus("completed");
      return;
    }
    setIndex((current) => current + 1);
  }

  function back() {
    if (index <= 0 && onBackTour) {
      onBackTour();
      return;
    }
    setIndex((current) => Math.max(0, current - 1));
  }

  return (
    <TourOverlay>
      <TourSpotlight rect={targetRect} />
      <TourPopover
        step={step}
        index={index}
        total={steps.length}
        position={position}
        onBack={back}
        onNext={next}
        onSkip={() => closeWithStatus("skipped")}
        backLabel={backLabel}
        nextLabel={nextLabel}
        doneLabel={doneLabel}
        skipLabel={skipLabel}
        displayIndex={displayIndex}
        displayTotal={displayTotal}
      />
    </TourOverlay>
  );
}
