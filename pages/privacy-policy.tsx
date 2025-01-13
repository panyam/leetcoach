import type { NextPage } from "next";
import Header from "../components/Header";
import PrivacyPolicy from "../components/PrivacyPolicy";
import styles from "../styles/Home.module.scss";

const Page: NextPage = () => {
  return (
    <main>
      <Header showLoginButton={true} hideHomeButton={false} styles={styles} showSearchButton ={true} />
      <PrivacyPolicy appName = "My Next Template" styles={styles} />
    </main>
  );
};

export default Page;
