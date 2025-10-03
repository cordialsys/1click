import { useState } from "react";
import Head from "next/head";
import ActivationPanel from "../components/ActivationPanel";

export default function Home() {
  return (
    <>
      <Head>
        <title>Treasury Panel - Cordial Systems</title>
        <link rel="icon" href="/images/logo-square-64.png" />
        <meta
          name="description"
          content="Treasury Panel Activation - Cordial Systems"
        />
      </Head>
      <div className="container">
        <div style={{ textAlign: "center", marginBottom: "2rem" }}>
          <img
            src="/images/logo-square-256-transparent.png"
            alt="Cordial Systems Logo"
            style={{
              height: "80px",
              width: "auto",
              marginBottom: "1rem",
            }}
          />
          <h1>Treasury Panel</h1>
        </div>
        <ActivationPanel />
      </div>
    </>
  );
}
