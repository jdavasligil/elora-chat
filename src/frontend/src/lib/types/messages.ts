interface Image {
  id: string;
  url: string;
  width: number;
  height: number;
}

interface Badge {
  name: string;
  title: string;
  icons: Image[];
  clickAction: string;
  clickURL: string;
}

export interface Emote {
  id: string;
  name: string;
  images: Image[];
  locations: unknown; // TODO: determine the correct type for this
}

export enum FragmentType {
  Text = "text",
  Emote = "emote",
  Colour = "colour",
  Effect = "effect",
  Pattern = "pattern",
}

export interface Fragment {
  type: FragmentType;
  text: string;
  emote: Emote;
}

export interface Message {
  author: string;
  badges: Badge[];
  colour: string;
  message: string;
  fragments: Fragment[];
  emotes: Emote[];
  source: 'YouTube' | 'Twitch';
}
