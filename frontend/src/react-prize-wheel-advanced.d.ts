// Type declarations for react-prize-wheel-advanced (the package ships no .d.ts)
declare module 'react-prize-wheel-advanced' {
    import * as React from 'react';

    export interface PrizeWheelProps {
        segments: string[];
        segColors: string[];
        onFinished?: (winnerIndex: number) => void;
        primaryColor?: string;
        contrastColor?: string;
        buttonText?: string;
        spinDuration?: number;
        size?: number;
    }

    export interface PrizeWheelHandle {
        click: () => void;
    }

    export const PrizeWheel: React.ForwardRefExoticComponent<
        PrizeWheelProps & React.RefAttributes<PrizeWheelHandle>
    >;

    export default PrizeWheel;
}
