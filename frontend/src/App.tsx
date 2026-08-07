import { useState } from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import { Greet, InitGame } from "../wailsjs/go/main/App";
import { Box, Button, Center, Heading, Image, Input, Text, VStack } from "@chakra-ui/react";
import { TimeWheel } from './components/TimeWheel';

function App() {
    const [resultText, setResultText] = useState("Please enter your name below 👇");
    const [name, setName] = useState('');
    const [gameStarted, setGameStarted] = useState(false);

    const updateName = (e: any) => setName(e.target.value);
    const updateResultText = (result: string) => setResultText(result);

    function handleStartGame(year: number) {
        InitGame(name || "Juliette", year).then((res: string) => {
            updateResultText(res);
            setGameStarted(true);
        });
    }

    return (
        <Center h="100vh" bg="gray.900" color="white">
            <VStack gap={8} p={8} bg="gray.800" borderRadius="xl" boxShadow="2xl" maxW="600px" w="full">
                <Image src={logo} h="120px" alt="logo" />
                
                <Heading size="lg" textAlign="center">{resultText}</Heading>
                
                {!gameStarted ? (
                    <VStack gap={6} w="full">
                        <Input 
                            placeholder="Enter your name (e.g. Juliette)" 
                            value={name} 
                            onChange={updateName} 
                            size="md"
                            bg="gray.700"
                            border="none"
                            _focus={{ border: "1px solid", borderColor: "blue.500" }}
                        />
                        <Text fontSize="sm" color="gray.400">
                            Spin the wheel to determine the starting point of your journey.
                        </Text>
                        <TimeWheel onSelect={handleStartGame} />
                    </VStack>
                ) : (
                    <VStack gap={4}>
                        <Button colorPalette="blue" onClick={() => setGameStarted(false)}>
                            Restart Initialization
                        </Button>
                    </VStack>
                )}

                <Box w="full" pt={4} borderTop="1px solid" borderColor="gray.700">
                    <Text color="gray.400" fontSize="sm" textAlign="center">
                        Silo40 Control Panel - Initializing World State
                    </Text>
                </Box>
            </VStack>
        </Center>
    )
}

export default App;
