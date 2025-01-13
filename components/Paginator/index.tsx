import React from "react";
import Image from "next/image";

type Props<T> = {
  currPage?: number;
  pageLength?: number;
  loader?: (offset: number, count: number) => Promise<T[]>;
  onLoaded?: (results: T[]) => void;
  onPrevious?: () => void,
  onNext?: () => void,
  styles?: any,
};

export class Paginator<T> extends React.Component<Props<T>> {
  state: any;
  prevPageButtonRef = React.createRef<HTMLButtonElement>();
  pageStatusSpanRef = React.createRef<HTMLSpanElement>();
  nextPageButtonRef = React.createRef<HTMLButtonElement>();

  constructor(props: any, context: any) {
    super(props, context);
    this.state = {
      hasMoreResults: false,
      currPage: props.currPage || 0,
      pageLength: props.pageLength || 50,
      showPrevious: props.showPrevious || false,
      showNext: props.showNext || false,
      enablePrevious: props.enablePrevious || false,
      enableNext: props.enableNext || false,
    }
  }
    
  get styles() {
    return this.props.styles || {};
  }

  render() {
    const styles = this.props.styles || {};
    const enabledStyle = (b: boolean) => b ? styles.enabled : styles.disabled
    return (
      <div className={styles.paginator}>
        {this.state.showPrevious ?
          <button className={`${styles.prevPageButton} ${enabledStyle(this.state.enablePrevious)}`}
                  onTouchEnd={() => { if (this.props.onPrevious) this.props.onPrevious()}}
                  onClick={() => { if (this.props.onPrevious) this.props.onPrevious()}}
                  ref={this.prevPageButtonRef} >&lt;</button>
          :
          null
        }
        <span ref={this.pageStatusSpanRef} className={styles.pageStatusSpan}>-</span>
        {this.state.showNext ?
          <button className={`${styles.nextPageButton} ${enabledStyle(this.state.enableNext)}`}
                  onTouchEnd={() => { if (this.props.onNext) this.props.onNext()}}
                  onClick={() => { if (this.props.onNext) this.props.onNext()}}
                  ref={this.nextPageButtonRef} >&gt;</button>
          :
          null
        }
      </div>
    );
  }

  onNextClicked() {
    if (this.props.onNext) {
      this.props.onNext();
    } else if (this.state.hasMoreResults) {
      this.load(this.state.currPage + 1);
    }
  }

  onPreviousClicked() {
    if (this.props.onPrevious) {
      this.props.onPrevious();
    } else {
      this.load(this.state.currPage - 1);
    }
  }

  load(page: number) {
    if (this.props.loader && page >= 0) {
      this.showPrevious(false, false);
      this.showNext(false, false);
      const pageLength = this.state.pageLength;
      this.props.loader(pageLength * page, pageLength + 1)
          .then((results: T[]) => {
        // First result is always the "Show all Alphabetically" link
        this.state.currPage = page;
        this.state.hasMoreResults = results.length > pageLength;
        this.setPage(page + 1);
        this.showPrevious(true, page > 0);
        this.showNext(true, this.state.hasMoreResults);
        if (this.props.onLoaded) {
          this.props.onLoaded(results);
        }
      });
    }
  }

  showNext(show: boolean|null, enable: boolean|null) {
    const updates = {} as any;
    if (show != null) {
      updates.showNext = show;
    }
    if (enable != null) {
      updates.enableNext = enable;
    }
    this.setState(ps => ({
      ...ps,
      ...updates,
    }));
  }

  showPrevious(show: boolean|null, enable: boolean|null) {
    const updates = {} as any;
    if (show != null) {
      updates.showPrevious = show;
    }
    if (enable != null) {
      updates.enablePrevious = enable;
    }
    this.setState(ps => ({
      ...ps,
      ...updates,
    }));
  }

  setPage(page = 1): void {
    if (this.pageStatusSpanRef.current != null) {
      this.pageStatusSpanRef.current.innerText = "" + page;
    }
  }

  /*
  enablePrevious(enabled = true) {
    if (this.prevPageButtonRef.current != null) {
      this.prevPageButtonRef.current.disabled = !enabled;
      this.prevPageButtonRef.current.style.display = enabled ? "inline" : "none";
      if (this.pageStatusSpanRef.current != null) {
        this.pageStatusSpanRef.current.style.display =
          this.prevPageButtonRef.current.disabled && this.nextPageButtonRef.current?.disabled ? "none" : "inline";
      }
    }
  }

  enableNext(enabled = true): void {
    if (this.nextPageButtonRef.current != null) {
      this.nextPageButtonRef.current.disabled = !enabled;
      this.nextPageButtonRef.current.style.display = enabled ? "inline" : "none";
      if (this.pageStatusSpanRef.current != null) {
        this.pageStatusSpanRef.current.style.display =
          this.nextPageButtonRef.current.disabled && this.nextPageButtonRef.current.disabled ? "none" : "inline";
      }
    }
  }
  */
}
