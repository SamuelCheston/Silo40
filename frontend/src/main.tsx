import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'
import { Provider } from './components/ui/provider'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <Provider>
            <App/>
        </Provider>
    </React.StrictMode>
)
