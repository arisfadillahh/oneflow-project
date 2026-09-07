import Script from 'next/script';
import '../globals.css';

export const metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL || 'https://oneflow.id'),
  title: 'AI CS WhatsApp untuk Bisnis Indonesia | Oneflow.id',
  description:
    'AI CS WhatsApp untuk membalas chat, follow-up pelanggan, mencatat order, booking, dan handoff admin dalam satu workspace operasional.',
  alternates: {
    canonical: '/',
  },
  openGraph: {
    title: 'AI CS WhatsApp untuk Bisnis Indonesia | Oneflow.id',
    description:
      'Balas chat, follow-up pelanggan, dan catat order dari satu workspace dengan AI yang tetap berada dalam kendali admin.',
    url: '/',
    siteName: 'Oneflow.id',
    locale: 'id_ID',
    type: 'website',
    images: [
      {
        url: '/brand/oneflow-main-logo.png',
        width: 512,
        height: 128,
        alt: 'Oneflow.id AI CS WhatsApp',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI CS WhatsApp untuk Bisnis Indonesia | Oneflow.id',
    description:
      'Balas chat, follow-up pelanggan, dan catat order dari satu workspace dengan AI yang tetap berada dalam kendali admin.',
    images: ['/brand/oneflow-main-logo.png'],
  },
  robots: {
    index: true,
    follow: true,
  },
};

const structuredData = {
  '@context': 'https://schema.org',
  '@graph': [
    {
      '@type': 'Organization',
      '@id': 'https://oneflow.id/#organization',
      name: 'PT ONEFLOW TEKNOLOGI NUSANTARA',
      url: 'https://oneflow.id/',
      logo: 'https://oneflow.id/brand/oneflow-main-logo.png',
      email: 'support@oneflow.id',
    },
    {
      '@type': 'SoftwareApplication',
      '@id': 'https://oneflow.id/#software',
      name: 'Oneflow.id',
      applicationCategory: 'BusinessApplication',
      operatingSystem: 'Web',
      description:
        'AI CS WhatsApp dan workspace operasional untuk membalas chat, follow-up pelanggan, order, booking, dan handoff admin.',
      url: 'https://oneflow.id/',
      publisher: { '@id': 'https://oneflow.id/#organization' },
    },
  ],
};

const hiddenUntilInteractionReady = `
  html.w-mod-js:not(.w-mod-ix3) :is([hero-load], [scroll-item="show"], .marquee-item, .accordion-content, .accordion-divider-vr, [button-icon-left], .button-icon-wrap.left, [cms-image="hover"], .accordion-card, [button-text], .integration-list, .integration-icon, .bento-marquee-list, [button-sm-text], [icon-sm-left], [list-item="show"], [btn-right-text], [btn-left], .dropdown-link-list, .icon, .comparison-item._01, .comparison-item._02, .stat-card._01, .stat-card._02.dark, .stat-card._03, .stat-card._04, .stat-card._05.dark, .feature-bottom-image._01, .feature-bottom-image._02, .bento-box-shadow, .hero-icon, .hero-image, .hero-decoration-image) {
    visibility: hidden;
  }
`;

const interactionRuntime = `
  (() => {
    const scripts = [
      "/oneflow_vendor/93_gsap.min.js",
      "/oneflow_vendor/3_ScrollTrigger.min.js",
      "/oneflow_vendor/58_lenis.min.js",
      "/oneflow_vendor/jquery-3.5.1.min.js",
      "/oneflow_vendor/32_webflow.schunk.66108cd89a639d26.js",
      "/oneflow_vendor/17_webflow.63ef20aa.d98eb2d8abb9c1d4.js",
    ];

    const initLenis = () => {
      if (typeof Lenis !== "undefined" && typeof gsap !== "undefined" && typeof ScrollTrigger !== "undefined") {
        const lenis = new Lenis({
          smooth: true,
          lerp: 0.1,
          wheelMultiplier: 1,
          infinite: false,
        });
        window.__oneflowLenis = lenis;
        lenis.on("scroll", ScrollTrigger.update);
        gsap.ticker.add((time) => {
          lenis.raf(time * 1000);
        });
        gsap.ticker.lagSmoothing(0);
      }
    };

    const initInternalNavigation = () => {
      if (window.__oneflowInternalNavigation) {
        return;
      }

      window.__oneflowInternalNavigation = true;
      document.addEventListener("click", (event) => {
        const link = event.target.closest('a[data-oneflow-scroll="setup"]');
        const targetId = link && link.getAttribute("href");
        const target = targetId && document.querySelector(targetId);

        if (!target) {
          return;
        }

        event.preventDefault();
        event.stopImmediatePropagation();
        window.history.pushState(null, "", targetId);

        const offset = window.innerWidth < 768 ? -102 : -126;
        const scrollToTarget = (duration) => {
          const destination = target.getBoundingClientRect().top + window.scrollY + offset;
          if (window.__oneflowLenis && typeof window.__oneflowLenis.scrollTo === "function") {
            window.__oneflowLenis.scrollTo(destination, { duration });
            return;
          }

          window.scrollTo({
            top: destination,
            behavior: "smooth",
          });
        };

        scrollToTarget(0.9);
        window.clearTimeout(window.__oneflowAnchorCorrection);
        window.__oneflowAnchorCorrection = window.setTimeout(() => {
          scrollToTarget(0.32);
        }, 980);
      }, true);
    };

    const initInteractiveHeroBackdrop = () => {
      const canvas = document.querySelector(".oneflow-hero-canvas");
      const section = document.querySelector(".hero-section");
      if (!canvas || !section) return;

      if (window.__oneflowHeroBackdropCleanup) {
        window.__oneflowHeroBackdropCleanup();
      }

      const context = canvas.getContext("2d");
      if (!context) return;

      const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
      let frameId = 0;
      let width = 0;
      let height = 0;
      let ratio = Math.min(window.devicePixelRatio || 1, 2);
      let spacing = 27;
      let pointerActive = false;
      const pointer = { x: 0, y: 0 };
      const smooth = { x: 0, y: 0 };

      const draw = (time) => {
        if (!width || !height) return;

        smooth.x += (pointer.x - smooth.x) * (reducedMotion ? 1 : 0.13);
        smooth.y += (pointer.y - smooth.y) * (reducedMotion ? 1 : 0.13);
        context.clearRect(0, 0, width, height);

        const glowRadius = pointerActive ? 230 : 188;
        const glow = context.createRadialGradient(
          smooth.x,
          smooth.y,
          0,
          smooth.x,
          smooth.y,
          glowRadius,
        );
        glow.addColorStop(0, "rgba(33, 150, 243, " + (pointerActive ? "0.22" : "0.10") + ")");
        glow.addColorStop(0.4, "rgba(91, 119, 255, " + (pointerActive ? "0.09" : "0.04") + ")");
        glow.addColorStop(1, "rgba(33, 150, 243, 0)");
        context.fillStyle = glow;
        context.fillRect(0, 0, width, height);

        const pulse = reducedMotion ? 0 : (Math.sin(time * 0.0014) + 1) * 0.012;
        const activeRadius = pointerActive ? 184 : 142;
        const activeRadiusSq = activeRadius * activeRadius;
        for (let y = spacing / 2; y < height; y += spacing) {
          for (let x = spacing / 2; x < width; x += spacing) {
            const dx = x - smooth.x;
            const dy = y - smooth.y;
            const distanceSq = dx * dx + dy * dy;
            let influence = 0;
            if (distanceSq < activeRadiusSq) {
              influence = 1 - Math.sqrt(distanceSq) / activeRadius;
              influence *= influence;
            }

            const alpha = 0.19 + pulse + influence * (pointerActive ? 0.49 : 0.1);
            const dotRadius = 1.24 + influence * (pointerActive ? 2.34 : 0.94);
            context.beginPath();
            context.fillStyle = influence > 0.25
              ? "rgba(33, 150, 243, " + Math.min(alpha, 0.7).toFixed(3) + ")"
              : "rgba(72, 143, 211, " + Math.min(alpha, 0.36).toFixed(3) + ")";
            context.arc(x, y, dotRadius, 0, Math.PI * 2);
            context.fill();
          }
        }
      };

      const resize = () => {
        const bounds = section.getBoundingClientRect();
        width = Math.max(1, Math.round(bounds.width));
        height = Math.max(1, Math.round(bounds.height));
        spacing = width < 768 ? 30 : 27;
        ratio = Math.min(window.devicePixelRatio || 1, 2);
        canvas.width = Math.round(width * ratio);
        canvas.height = Math.round(height * ratio);
        canvas.style.width = width + "px";
        canvas.style.height = height + "px";
        context.setTransform(ratio, 0, 0, ratio, 0, 0);
        if (!pointerActive) {
          pointer.x = width * 0.5;
          pointer.y = Math.min(height * 0.27, 330);
          smooth.x = pointer.x;
          smooth.y = pointer.y;
        }
        draw(0);
      };

      const move = (event) => {
        const bounds = section.getBoundingClientRect();
        pointer.x = event.clientX - bounds.left;
        pointer.y = event.clientY - bounds.top;
        pointerActive = true;
      };

      const leave = () => {
        pointerActive = false;
        pointer.x = width * 0.5;
        pointer.y = Math.min(height * 0.27, 330);
      };

      const tick = (time) => {
        draw(time);
        frameId = window.requestAnimationFrame(tick);
      };

      const observer = typeof ResizeObserver !== "undefined" ? new ResizeObserver(resize) : null;
      if (observer) observer.observe(section);
      window.addEventListener("resize", resize);
      section.addEventListener("pointermove", move, { passive: true });
      section.addEventListener("pointerleave", leave);
      resize();
      if (!reducedMotion) {
        frameId = window.requestAnimationFrame(tick);
      }

      window.__oneflowHeroBackdropCleanup = () => {
        window.cancelAnimationFrame(frameId);
        if (observer) observer.disconnect();
        window.removeEventListener("resize", resize);
        section.removeEventListener("pointermove", move);
        section.removeEventListener("pointerleave", leave);
        window.__oneflowHeroBackdropCleanup = null;
      };
    };

    const initScrollComparison = () => {
      const section = document.querySelector(".oneflow-scroll-comparison");
      const sticky = section && section.querySelector(".oneflow-compare-sticky");
      const frame = section && section.querySelector("[data-oneflow-compare]");
      if (!section || !sticky || !frame) {
        return;
      }

      if (window.__oneflowComparisonMotion) {
        window.__oneflowComparisonMotion.revert();
        window.__oneflowComparisonMotion = null;
      }

      if (window.__oneflowComparisonCleanup) {
        window.__oneflowComparisonCleanup();
      }

      const controls = Array.from(
        frame.querySelectorAll("[data-comparison-view]"),
      );

      const setProgress = (progress) => {
        const value = Math.max(0, Math.min(100, progress));
        const afterActive = value >= 74;
        frame.style.setProperty("--comparison-progress", value + "%");
        frame.classList.toggle("is-after-active", afterActive);
        controls.forEach((control) => {
          const pressed =
            control.dataset.comparisonView === (afterActive ? "after" : "before");
          control.setAttribute("aria-pressed", String(pressed));
        });
      };

      const chooseView = (event) => {
        setProgress(
          event.currentTarget.dataset.comparisonView === "after" ? 100 : 0,
        );
      };

      controls.forEach((control) => control.addEventListener("click", chooseView));
      window.__oneflowComparisonCleanup = () => {
        controls.forEach((control) =>
          control.removeEventListener("click", chooseView),
        );
        window.__oneflowComparisonCleanup = null;
      };

      if (
        window.matchMedia("(max-width: 767px)").matches ||
        window.matchMedia("(prefers-reduced-motion: reduce)").matches ||
        typeof gsap === "undefined" ||
        typeof ScrollTrigger === "undefined"
      ) {
        setProgress(0);
        return;
      }

      gsap.registerPlugin(ScrollTrigger);
      setProgress(0);
      window.__oneflowComparisonMotion = gsap.context(() => {
        const state = { progress: 0 };
        gsap.to(state, {
          progress: 100,
          ease: "none",
          onUpdate: () => setProgress(state.progress),
          scrollTrigger: {
            trigger: section,
            start: "top top+=72",
            end: "+=112%",
            scrub: 0.35,
            pin: section,
            pinSpacing: true,
            pinReparent: true,
            anticipatePin: 1,
            invalidateOnRefresh: true,
          },
        });
      }, section);
    };

    const initSetupSteps = () => {
      const section = document.querySelector(".oneflow-setup-section");
      const stage = section && section.querySelector("[data-oneflow-setup]");
      if (!section || !stage) {
        return;
      }

      if (window.__oneflowSetupMotion) {
        window.__oneflowSetupMotion.revert();
        window.__oneflowSetupMotion = null;
      }

      const progressItems = Array.from(
        stage.querySelectorAll("[data-setup-step]"),
      );
      const panels = Array.from(stage.querySelectorAll("[data-setup-panel]"));
      const setStep = (step) => {
        const activeStep = Math.max(0, Math.min(2, step));
        stage.classList.toggle("is-step-2", activeStep === 1);
        stage.classList.toggle("is-step-3", activeStep === 2);
        progressItems.forEach((item, index) => {
          item.classList.toggle("is-active", index === activeStep);
          if (index === activeStep) {
            item.setAttribute("aria-current", "step");
          } else {
            item.removeAttribute("aria-current");
          }
        });
        panels.forEach((panel, index) => {
          panel.classList.toggle("is-active", index === activeStep);
        });
      };

      setStep(0);
      if (
        window.matchMedia("(max-width: 991px)").matches ||
        window.matchMedia("(prefers-reduced-motion: reduce)").matches ||
        typeof gsap === "undefined" ||
        typeof ScrollTrigger === "undefined"
      ) {
        return;
      }

      gsap.registerPlugin(ScrollTrigger);
      window.__oneflowSetupMotion = gsap.context(() => {
        const state = { progress: 0 };
        gsap.to(state, {
          progress: 1,
          ease: "none",
          onUpdate: () => setStep(Math.min(2, Math.floor(state.progress * 3))),
          scrollTrigger: {
            trigger: section,
            start: "top top+=82",
            end: "+=150%",
            scrub: 0.32,
            pin: section,
            pinSpacing: true,
            pinReparent: true,
            anticipatePin: 1,
            invalidateOnRefresh: true,
          },
        });
      }, section);
    };

    const initHeroDashboardParallax = () => {
      if (
        typeof gsap === "undefined" ||
        typeof ScrollTrigger === "undefined" ||
        window.matchMedia("(prefers-reduced-motion: reduce)").matches ||
        window.__oneflowHeroDashboardParallax
      ) {
        return;
      }

      gsap.registerPlugin(ScrollTrigger);
      window.__oneflowHeroDashboardParallax = gsap.context(() => {
        if (!document.querySelector(".hero-decoration-image")) return;
        gsap.to(".hero-decoration-image", {
          yPercent: -8,
          ease: "none",
          scrollTrigger: {
            trigger: ".hero-section",
            start: "top top",
            end: "bottom top",
            scrub: 0.7,
          },
        });
      });
    };

    const load = (index) => {
      if (index >= scripts.length) {
        initLenis();
        initInternalNavigation();
        initHeroDashboardParallax();
        initScrollComparison();
        initSetupSteps();
        return;
      }
      const script = document.createElement("script");
      script.src = scripts[index];
      script.async = false;
      script.onload = () => load(index + 1);
      script.onerror = () => load(index + 1);
      document.body.appendChild(script);
    };

    initInteractiveHeroBackdrop();
    load(0);
  })();
`;

export default function RootLayout({ children }) {
  return (
    <html
      lang="id"
      data-wf-domain="oneflow.id"
      data-wf-page="69b2a2adca3cdacc51788e3b"
      data-wf-site="69b2a2adca3cdacc51788e5b"
      suppressHydrationWarning
    >
      <head>
        <link rel="icon" href="/icon.png" type="image/png" />
        <link
          href="/oneflow_vendor/oneflow.webflow.shared.52d73283f.min.css"
          rel="stylesheet"
          type="text/css"
          crossOrigin="anonymous"
        />
        <link rel="stylesheet" href="/oneflow_vendor/98_lenis.css" />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(structuredData) }}
        />
        <style dangerouslySetInnerHTML={{ __html: hiddenUntilInteractionReady }} />
      </head>
      <body suppressHydrationWarning>
        {children}
        <Script id="oneflow-interaction-runtime" strategy="afterInteractive">
          {interactionRuntime}
        </Script>
      </body>
    </html>
  );
}
