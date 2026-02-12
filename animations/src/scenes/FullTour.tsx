import React from "react";
import { TransitionSeries, linearTiming } from "@remotion/transitions";
import { fade } from "@remotion/transitions/fade";

import { TitleCard } from "./TitleCard";
import { Scene1RegisterDomain } from "./Scene1RegisterDomain";
import { Scene2DnsVerification } from "./Scene2DnsVerification";
import { Scene3EmbedWidget } from "./Scene3EmbedWidget";
import { Scene4Dashboard } from "./Scene4Dashboard";
import { OutroCard } from "./OutroCard";

/**
 * FullTour -- Complete product demo combining all scenes with fade
 * transitions via TransitionSeries.
 *
 * Duration: 645 frames (21.5s @ 30fps)
 *
 * Calculation:
 *   60 + 135 + 165 + 150 + 150 + 60 = 720 total sequence frames
 *   5 transitions x 15 frames each  =  75 overlapping frames
 *   720 - 75 = 645 net frames
 */
export const FullTour: React.FC = () => {
  return (
    <TransitionSeries>
      <TransitionSeries.Sequence durationInFrames={60}>
        <TitleCard />
      </TransitionSeries.Sequence>

      <TransitionSeries.Transition
        presentation={fade()}
        timing={linearTiming({ durationInFrames: 15 })}
      />

      <TransitionSeries.Sequence durationInFrames={135}>
        <Scene1RegisterDomain />
      </TransitionSeries.Sequence>

      <TransitionSeries.Transition
        presentation={fade()}
        timing={linearTiming({ durationInFrames: 15 })}
      />

      <TransitionSeries.Sequence durationInFrames={165}>
        <Scene2DnsVerification />
      </TransitionSeries.Sequence>

      <TransitionSeries.Transition
        presentation={fade()}
        timing={linearTiming({ durationInFrames: 15 })}
      />

      <TransitionSeries.Sequence durationInFrames={150}>
        <Scene3EmbedWidget />
      </TransitionSeries.Sequence>

      <TransitionSeries.Transition
        presentation={fade()}
        timing={linearTiming({ durationInFrames: 15 })}
      />

      <TransitionSeries.Sequence durationInFrames={150}>
        <Scene4Dashboard />
      </TransitionSeries.Sequence>

      <TransitionSeries.Transition
        presentation={fade()}
        timing={linearTiming({ durationInFrames: 15 })}
      />

      <TransitionSeries.Sequence durationInFrames={60}>
        <OutroCard />
      </TransitionSeries.Sequence>
    </TransitionSeries>
  );
};
