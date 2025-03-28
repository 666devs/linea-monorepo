import { Direction } from "../../enums/MessageEnums";

export interface IL2ClaimMessageTransactionSizeProcessor {
  process(): Promise<void>;
}

export type L2ClaimMessageTransactionSizeProcessorConfig = {
  direction: Direction;
  originContractAddress: string;
};
