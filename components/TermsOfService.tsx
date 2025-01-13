
const TermsOfService = (props: any) => {
    return (
      <>
    <h2 id="1-terms">1. Terms</h2>

    <p>By accessing our app, {props.appName}, you are agreeing to be bound by these terms of service, all applicable laws and regulations, and agree that you are responsible for compliance with any applicable local laws. If you do not agree with any of these terms, you are prohibited from using or accessing {props.appName}. The materials contained in {props.appName} are protected by applicable copyright and trademark law.</p>

    <h2 id="2-use-license">2. Use License</h2>

    <ol>
      <li>Permission is granted to temporarily download multiple copies of {props.appName} per device for personal, non-commercial transitory viewing only. This is the grant of a license, not a transfer of title, and under this license you may not:
        <ol>
          <li>use the materials for any commercial purpose, or for any public display (commercial or non-commercial);</li>
          <li>remove any copyright or other proprietary notations from the materials;</li>
        </ol>
      </li>
      <li>This license shall automatically terminate if you violate any of these restrictions and may be terminated by {props.appName} at any time. Upon terminating your viewing of these materials or upon the termination of this license, you must destroy any downloaded materials in your possession whether in electronic or printed format.</li>
    </ol>

    <h2 id="1-disclaimer">1. Disclaimer</h2>

    <ol>
      <li>
        <p>The materials within {props.appName} are provided on an ‘as is’ basis. {props.appName} makes no warranties, expressed or implied, and hereby disclaims and negates all other warranties including, without limitation, implied warranties or conditions of merchantability, fitness for a particular purpose, or non-infringement of intellectual property or other violation of rights.</p>
      </li>
      <li>
        <p>Further, {props.appName} does not warrant or make any representations concerning the accuracy, likely results, or reliability of the use of the materials on its website or otherwise relating to such materials or on any sites linked to {props.appName}.</p>
      </li>
    </ol>

    <h2 id="4-limitations">4. Limitations</h2>

    <p>In no event shall {props.appName} or its suppliers be liable for any damages (including, without limitation, damages for loss of data or profit, or due to business interruption) arising out of the use or inability to use {props.appName}, even if {props.appName} or a {props.appName} authorized representative has been notified orally or in writing of the possibility of such damage. Because some jurisdictions do not allow limitations on implied warranties, or limitations of liability for consequential or incidental damages, these limitations may not apply to you.</p>

    <h2 id="5-accuracy-of-materials">5. Accuracy of materials</h2>

    <p>The materials appearing in {props.appName} could include technical, typographical, or photographic errors. {props.appName} does not warrant that any of the materials on {props.appName} are accurate, complete or current. {props.appName} may make changes to the materials contained in {props.appName} at any time without notice. However {props.appName} does not make any commitment to update the materials.</p>

    <h2 id="6-links">6. Links</h2>

    <p>{props.appName} has not reviewed all of the sites linked to its app and is not responsible for the contents of any such linked site. The inclusion of any link does not imply endorsement by {props.appName} of the site. Use of any such linked website is at the user’s own risk.</p>

    <h2 id="7-modifications">7. Modifications</h2>

    <p>{props.appName} may revise these terms of service for its app at any time without notice. By using {props.appName} you are agreeing to be bound by the then current version of these terms of service.</p>

    <h2 id="8-governing-law">8. Governing Law</h2>

    <p>These terms and conditions are governed by and construed in accordance with the laws of California and you irrevocably submit to the exclusive jurisdiction of the courts in that State or location.</p>
  </>
  )
}

export default TermsOfService;
