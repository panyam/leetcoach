import type { NextPage } from "next";
import Header from "../components/Header";
import TermsOfService from "../components/TermsOfService";
import styles from "../styles/Home.module.scss";

const Page: NextPage = () => {
  return (
    <main>
      <Header showLoginButton={true} hideHomeButton={false} styles={styles}  showSearchButton ={true} />
      <TermsOfService appName = "My Next Template" styles={styles} />
    </main>
  );
};

export default Page;
