import "./index.css";
import { Composition, Folder } from "remotion";
import { VIDEO } from "./constants/theme";
import { Scene1RegisterDomain } from "./scenes/Scene1RegisterDomain";
import { Scene2DnsVerification } from "./scenes/Scene2DnsVerification";
import { Scene3EmbedWidget } from "./scenes/Scene3EmbedWidget";
import { Scene4Dashboard } from "./scenes/Scene4Dashboard";
import { TitleCard } from "./scenes/TitleCard";
import { OutroCard } from "./scenes/OutroCard";
import { FullTour } from "./scenes/FullTour";

export const RemotionRoot: React.FC = () => {
  return (
    <>
      <Folder name="Scenes">
        <Composition
          id="Scene1-RegisterDomain"
          component={Scene1RegisterDomain}
          durationInFrames={135}
          fps={VIDEO.fps}
          width={VIDEO.width}
          height={VIDEO.height}
        />
        <Composition
          id="Scene2-DnsVerification"
          component={Scene2DnsVerification}
          durationInFrames={165}
          fps={VIDEO.fps}
          width={VIDEO.width}
          height={VIDEO.height}
        />
        <Composition
          id="Scene3-EmbedWidget"
          component={Scene3EmbedWidget}
          durationInFrames={150}
          fps={VIDEO.fps}
          width={VIDEO.width}
          height={VIDEO.height}
        />
        <Composition
          id="Scene4-Dashboard"
          component={Scene4Dashboard}
          durationInFrames={150}
          fps={VIDEO.fps}
          width={VIDEO.width}
          height={VIDEO.height}
        />
      </Folder>
      <Folder name="Cards">
        <Composition
          id="TitleCard"
          component={TitleCard}
          durationInFrames={60}
          fps={VIDEO.fps}
          width={VIDEO.width}
          height={VIDEO.height}
        />
        <Composition
          id="OutroCard"
          component={OutroCard}
          durationInFrames={60}
          fps={VIDEO.fps}
          width={VIDEO.width}
          height={VIDEO.height}
        />
      </Folder>
      <Composition
        id="FullTour"
        component={FullTour}
        durationInFrames={645}
        fps={VIDEO.fps}
        width={VIDEO.width}
        height={VIDEO.height}
      />
    </>
  );
};
