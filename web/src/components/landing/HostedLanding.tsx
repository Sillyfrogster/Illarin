import { ArrowDown, ArrowRight, ArrowUpRight, RotateCcw } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { LandingMotion } from "./LandingMotion";
import "./journey.css";

export function HostedLanding({ children }: { children: ReactNode }) {
  return (
    <LandingMotion>
      <div
        id="journey"
        data-composition-id="illarin-flight"
        data-duration="100"
      >
        <div className="stage">
          <canvas id="world" aria-hidden="true" tabIndex={-1}></canvas>
          <div className="vignette" aria-hidden="true"></div>
          <div className="film-grain" aria-hidden="true"></div>
          <section
            id="beginning"
            className="chapter opening"
            aria-labelledby="opening-title"
          >
            <Image
              className="still-art"
              src="/landing/flight/room-start.webp"
              alt="The first few pencil strokes of a character on an otherwise blank notebook page."
              width={1920}
              height={1080}
              unoptimized
            />
            <div className="copy">
              <p className="chapter-note">A home for your imagination.</p>
              <h1 id="opening-title">
                It starts with
                <br />
                <em>an idea.</em>
              </h1>
              <p className="description">
                A character. A place. A story only you could tell.
              </p>
            </div>
          </section>
          <section
            id="worlds"
            className="chapter kingdom"
            aria-labelledby="kingdom-title"
          >
            <Image
              className="still-art"
              src="/landing/flight/kingdom.webp"
              alt="A stone kingdom above a river, with mountains disappearing into violet dusk."
              loading="lazy"
              width={1920}
              height={1080}
              unoptimized
            />
            <div className="copy">
              <p className="chapter-note">Make room for the details.</p>
              <h2 id="kingdom-title">
                Give it a voice.
                <br />
                Give it <em>a world.</em>
              </h2>
              <p className="description">
                Build characters and lorebooks around the stories you want to
                tell.
              </p>
            </div>
          </section>
          <section
            id="possibilities"
            className="chapter city"
            aria-labelledby="city-title"
          >
            <Image
              className="still-art"
              src="/landing/flight/city.webp"
              alt="A futuristic city of violet lights and sky bridges, reflected in the river below."
              loading="lazy"
              width={1920}
              height={1080}
              unoptimized
            />
            <div className="copy">
              <p className="chapter-note">There’s more than one way to play.</p>
              <h2 id="city-title">
                Take your stories
                <br />
                <em>somewhere new.</em>
              </h2>
              <p className="description">
                Share what you make. Bring it into the roleplay apps you already
                love.
              </p>
            </div>
          </section>
          <section
            id="creators"
            className="chapter gallery"
            aria-labelledby="gallery-title"
          >
            <Image
              className="still-art"
              src="/landing/flight/gallery.webp"
              alt="A quiet moonlit gallery overlooking the city."
              loading="lazy"
              width={1920}
              height={1080}
              unoptimized
            />
            <div className="gallery-heading">
              <div>
                <p className="chapter-note">
                  From the people who make Illarin.
                </p>
                <h2 id="gallery-title">
                  Meet your <em>next story.</em>
                </h2>
              </div>
              <p className="gallery-intro">
                Characters, worlds, and the people behind them.
              </p>
            </div>
            <div className="gallery-works">{children}</div>
            <div className="gallery-caption">
              <span>Recently published on Illarin</span>
              <Link href="/browse">
                Browse creations <ArrowUpRight aria-hidden="true" size={18} />
              </Link>
            </div>
          </section>
          <section
            id="homecoming"
            className="chapter ending"
            aria-labelledby="ending-title"
          >
            <Image
              className="still-art"
              src="/landing/flight/room-finished.webp"
              alt="The notebook now holds a finished, richly painted character in a violet cloak."
              loading="lazy"
              width={1920}
              height={1080}
              unoptimized
            />
            <div className="copy">
              <p className="chapter-note">
                From the first idea to whatever comes next.
              </p>
              <h2 id="ending-title" className="ending-wordmark">
                illarin<span>.</span>
              </h2>
              <p className="description">
                A home for the things you make and the people you make them
                with.
              </p>
              <div className="actions">
                <Button asChild variant="primary" size="large">
                  <Link href="/upload">
                    Start creating <ArrowUpRight aria-hidden="true" />
                  </Link>
                </Button>
                <Button asChild variant="ghost" size="large">
                  <Link href="/browse">
                    Browse creations <ArrowRight aria-hidden="true" />
                  </Link>
                </Button>
              </div>
              <Button variant="ghost" size="compact" className="replay">
                <RotateCcw aria-hidden="true" /> Replay the journey
              </Button>
            </div>
          </section>
          <div className="arrival" aria-hidden="true"></div>
          <div className="journey-footer">
            <Button variant="ghost" size="compact" className="follow">
              <ArrowDown className="forward-icon" aria-hidden="true" />
              <RotateCcw className="return-icon" aria-hidden="true" />
              <span id="instruction">Follow the butterflies</span>
            </Button>
            <nav className="chapters" aria-label="Journey chapters">
              <Link
                href="#beginning"
                aria-label="The beginning"
                aria-current="step"
              >
                <span>01</span>
              </Link>
              <Link href="#worlds" aria-label="Make a world">
                <span>02</span>
              </Link>
              <Link href="#possibilities" aria-label="Go further">
                <span>03</span>
              </Link>
              <Link href="#creators" aria-label="Creator showcase">
                <span>04</span>
              </Link>
              <Link href="#homecoming" aria-label="Welcome to Illarin">
                <span>05</span>
              </Link>
              <div className="chapter-line" aria-hidden="true">
                <i></i>
              </div>
            </nav>
          </div>
        </div>
      </div>
    </LandingMotion>
  );
}
