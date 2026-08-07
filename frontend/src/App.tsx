import { useState } from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import { Greet } from "../wailsjs/go/main/App";
import { Box, Button, Center, Heading, Image, Input, Stack, Text, VStack } from "@chakra-ui/react";

function App() {
    const [resultText, setResultText] = useState("Please enter your name below 👇");
    const [name, setName] = useState('');
    const updateName = (e: any) => setName(e.target.value);
    const updateResultText = (result: string) => setResultText(result);

    function greet() {
        Greet(name).then(updateResultText);
    }

    return (
        <Center h="100vh" bg="gray.900" color="white">
            <VStack gap={8} p={8} bg="gray.800" borderRadius="xl" boxShadow="2xl">
                <Image src={logo} h="200px" alt="logo" />
                
                <Heading size="lg">{resultText}</Heading>
                
                <Stack direction="row" gap={4} w="full">
                    <Input 
                        placeholder="Enter your name" 
                        value={name} 
                        onChange={updateName} 
                        size="md"
                        bg="gray.700"
                        border="none"
                        _focus={{ border: "1px solid", borderColor: "blue.500" }}
                    />
                    <Button colorPalette="blue" onClick={greet} px={8}>
                        Greet
                    </Button>
                </Stack>

                <Box w="full" pt={4} borderTop="1px solid" borderColor="gray.700">
                    <Text color="gray.400" fontSize="sm">
                        Silo40 Control Panel
                    </Text>
                </Box>
            </VStack>
        </Center>
    )
}

export default App
