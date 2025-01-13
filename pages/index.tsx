import React from "react";
import Head from "next/head";
import Header from "../components/Header";
import styles from "../styles/Home.module.scss";
import BaseComponent from "../components/BaseComponent";

export default class Home extends BaseComponent<any>{
  state: any;
  constructor(props: any, context: any) {
    super(props, context)
    this.state = {
    };
  }

  render() {
    return (
      <main>
        <Header showLoginButton={true} hideHomeButton={true} showLogoutButton={true}
                styles={styles} showSearchButton ={true}
                extraMenuItems={[]} />
        Hello world - Your starter NextJS Template.
      </main>
    );
  }

  componentDidMount() {
  }
}
