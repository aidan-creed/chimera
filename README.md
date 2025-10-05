# Chimera

An open-source platform powering an AI-driven assisted analytics service for EVE Online.

---

## Overview

`Chimera` will provide a suite of tools designed to help capsuleers of all experience levels navigate the complexities of New Eden. Our mission is to go beyond simple data grids by providing an **assisted analytics** service that acts as an intelligent co-pilot, transforming vast market, combat, and industrial data into clear, actionable insights. This approach empowers players to make their own smarter, more profitable decisions with confidence.

This project was born from the desire to fuse modern AI/ML workflows with the rich, complex data streams of the EVE Online universe, creating a service that is both powerful for experts and accessible to new players. 

The services provided by `Chimera` will utilize game & character data from the ESI, Eve's Static Data Export & zKillboard. To learn how we use your data, read our [PRIVACY POLICY](privacy_policy_link)

## Core Principles

The `Chimera` platform is built on a set of pragmatic, modern architectural principles:

* **Modular Monolith First:** The backend is a single, deployable Go application composed of highly modular, decoupled packages. This maximizes initial velocity while providing a clear path to extract microservices as needed.
* **Assisted Analytics:** Our goal is to empower the user. We leverage AI to synthesize information and surface insights, acting as an intelligent aid that assists the user in making their own informed decisions.
* **API-First and Decoupled:** The platform is built around a clean, versioned API that serves a modern, client-side rendered frontend. This provides a clear separation of concerns and enables future integrations.

## Technology Stack

The platform uses a curated stack of modern, robust, and open-source technologies.

### Backend
* **Language:** Go
* **Framework:** Echo
* **Database:** PostgreSQL with the `pgvector` extension
* **Migrations:** goose
* **Queries:** sqlc for type-safe query generation
* **Deployment:** Docker, GCP

### Frontend
* **Framework:** React with TypeScript
* **Build Tool:** Vite
* **Styling:** Tailwind CSS
* **Components:** `shadcn/ui` (built on Radix UI)
* **Data Grids:** TanStack Table

## Getting Started

*(This section will be built out as the project matures)*

### Prerequisites

* Go (version X.X)
* Node.js (version X.X)
* Docker

### Installation

1.  Clone the repository:
    ```bash
    git clone [Your New Chimera Repo URL]
    ```
2.  Install backend dependencies:
    ```bash
    cd chimera/backend && go mod tidy
    ```
3.  Install frontend dependencies:
    ```bash
    cd ../frontend && npm install
    ```

### Running Locally

*(Detailed instructions for setting up local environment variables and running the services will be added to a `DEVELOPING.md` guide.)*

## Roadmap

We are working on a pre-release version of the service we are calling **Foundation**. It is a suite of high-value, free-to-use analytics tools based on public data. The goal is to build our brand on a foundation of trust and provide immediate value to the EVE community.

When **Foundation** is complete we will explore how to build on it.
