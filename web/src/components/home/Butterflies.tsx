import type { CSSProperties } from "react";
import "./home.css";

/** FLIGHTS is where each butterfly leaves her hand, how big it is, how far it travels and how long it takes. */
const FLIGHTS = [
  {
    x: "57%",
    y: "34%",
    size: "2.2rem",
    dx: "-22rem",
    dy: "-9rem",
    time: "11s",
    delay: "0s",
  },
  {
    x: "60%",
    y: "38%",
    size: "1.6rem",
    dx: "-30rem",
    dy: "-6rem",
    time: "14s",
    delay: "-4s",
  },
  {
    x: "55%",
    y: "30%",
    size: "1.9rem",
    dx: "-16rem",
    dy: "-12rem",
    time: "12s",
    delay: "-8s",
  },
  {
    x: "58%",
    y: "40%",
    size: "1.4rem",
    dx: "-26rem",
    dy: "-14rem",
    time: "16s",
    delay: "-2s",
  },
  {
    x: "61%",
    y: "33%",
    size: "1.7rem",
    dx: "-12rem",
    dy: "-10rem",
    time: "10s",
    delay: "-6s",
  },
];

/** Butterflies carries the painted stream on: a few live ones keep leaving her hand and fade toward the line. */
export function Butterflies() {
  return FLIGHTS.map((flight) => (
    <span
      className="home-butterfly"
      key={flight.delay}
      style={
        {
          "--x": flight.x,
          "--y": flight.y,
          "--size": flight.size,
          "--dx": flight.dx,
          "--dy": flight.dy,
          "--time": flight.time,
          "--delay": flight.delay,
        } as CSSProperties
      }
    >
      <i />
      <i />
    </span>
  ));
}
