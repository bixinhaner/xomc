<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style>
	#compactParamConfig{
		position:relative;
		height:80vh;
	}
	.cellSetting_basic_header{
	    width: 100%;
	    line-height: 50px;
	    height: 50px;
	    color: #7993B6;
	    font-size: 16px;
	}
	.clearBoth{
		float:none !important;
		clear:both;
		margin-right:400px !important;
		/* height:50px !important; */
	}
	.itemDiv{
		width:375px;
		height:92px;
		float:left;
		margin-right:45px;
	} 
	.promptTitle{
		height:24px;
		line-height:24px;
		min-width:50px;
		font-size:12px;
		color:#9FB318;
		display : none;
	}
	.errorTitle{
		height:24px;
		line-height:24px;
		min-width:50px;
		font-size:12px;
		color:red;
		display : none;
	}
	#compactParamConfig .itemDiv input,#compactParamConfig .itemDiv select{
		width:350px;
	}
	#compactParamConfig .itemDiv img{
		position:relative;
		top:4px;
		left:5px;
	}
	.omcTabsPage > div{
		height:unset;
	}
	.divideLine{
		max-width:800px;
		width:770px;
		margin: 35px 0 20px;
		border: 1px solid #DCECF7;
		clear : both;
	}
	.enbCountExportTip{
		display:inline-block;
		font-weight:700;
		font-size: 14px;
		color: #0f344d;
		margin-left:5px;
	}
	.divideLineTitle{
		margin-bottom:20px;
	}
	#compactParamConfig select:disabled{
		background:#eaf1f4;
	}
</style>

<!-- header -->
<div class="cellSetting_basic_header">
	<div class="cellSettingTitle_basic" style="display:inline-block;margin-left:30px;"><%=rb.getString("SheZhi")%></div>
	<a class="titleIcon_close iconSize" style="float:right;margin-top:15px;margin-right:10px;cursor:pointer;" onclick="cellSettingCancel()"></a>
</div>
	
<div id="compactParamConfig" class="inputInfos">
	<!-- 选项 -->
	<div class="omcPageTitleDiv omcLogLists">
		<ul class="omcPageTitleContainer titleTabsList">
			<li tabtit="wirelessSetting" onclick="turnTabs(this)" class="active"><%=rb.getString("WuXianSheZhi")%></li>
			<li tabtit="intentSetting" onclick="turnTabs(this)"><%=rb.getString("WangLuoSheZhi")%></li>
			<li tabtit="NTPSetting" onclick="turnTabs(this)"><%=rb.getString("NTPSheZhi")%></li>
			<c:if test="${lgw == 1}">
				<li tabtit="LGWSetting" onclick="turnTabs(this)"><%=rb.getString("LGWSheZhi")%></li>
			</c:if>
		</ul>
	</div>
	<div class="omcTabsPage logsTabsMainPage" style="left:20px;top:45px;overflow-x:hidden;overflow-y:auto;height:60vh;">
		<%-- 无线设置面板 --%>
		<div class="wirelessSetting" style="padding: 20px 0 0 20px;display:block;">
			<div class="itemDiv">
				<span title="<%=rb.getString("RSCellName")%>"><%=rb.getString("HostName")%></span>
				<input type="text" name="LTE_HOME_NODEB_NAME" id="LTE_HOME_NODEB_NAME" class="border border-box item" oldValue="${HOST_NAME}" value="${HOST_NAME}"
						onblur="validateHostName(event)" max_length="48"
						title="<%=rb.getString("SheBeiMingChengGuiZe")%>"/>
				<div class="errorTitle" id="LTE_HOME_NODEB_NAME_err"><%=rb.getString("SheBeiMingChengGuiZe")%></div>
			</div> 
			<div class="itemDiv">
				<span title="<%=rb.getString("RSMME")%>"><%=rb.getString("HeXinWang")%></span>
				<input type="text" name="LTE_SIGLINK_SERVER_LIST" id="LTE_SIGLINK_SERVER_LIST" class="border border-box item" 
					oldValue="" value="" onblur="validateIPAddress1(event)" 
					title="<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>"/>
				<%-- <a onclick="addMMEIpInputText(this)">
					<img src="${ctx}/skin/${manufacturer}/images/bi/setting_add.png"/>  
				</a> --%>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				<div class="errorTitle" id="LTE_SIGLINK_SERVER_LIST_err"><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></div>
			</div>
			<div class="divideLine"></div>
			<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("XiaoQuCanShu")%> 1</span>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSECI")%>"><%=rb.getString("JiZhanID")%></span>
				<input id="LTE_CELL_ECI_1" type="text" name="LTE_CELL_ECI_1" class="border border-box item" oldValue="${CELL_IDENTITY_1}" value="${CELL_IDENTITY_1}"
						onblur="validateMaxAndMinVal1(event)" min_value="0" max_value="268435455" 
						title="int, min value: 0, max value: 268435455"/>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				<div class="errorTitle" id="LTE_CELL_ECI_1_err">int, min value: 0, max value: 268435455</div>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSPLMN")%>"><%=rb.getString("PLMN")%></span>
				<input type="text" name="LTE_OAM_PLMNID_1" id="LTE_OAM_PLMNID_1" class="border border-box item" oldValue="${PLMNID_1}" value="${PLMNID_1}"
						onblur="validateMaxAndMinVal1(event)" min_value="10000" max_value="999999"
						title="<%=rb.getString("PLMNTitle")%>"/>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
				<div class="errorTitle" id="LTE_OAM_PLMNID_1_err"><%=rb.getString("PLMNTitle")%></div>
			</div>
			<div class="itemDiv">
				<c:if test="${isElfCell != 1}">
					<span title="<%=rb.getString("RSPCI")%>"><%=rb.getString("PCI2")%></span>
					<input type="text" name="LTE_PHY_CELLID_LIST_1" id="LTE_PHY_CELLID_LIST_1" class="border border-box item" oldValue="${PCI_1}" value="${PCI_1}"
						onblur="validateMaxAndMinVal1(event)" min_value="0" max_value="503"
						title="int, min value: 0, max value: 503" />
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
					<div class="errorTitle" id="LTE_PHY_CELLID_LIST_1_err">int, min value: 0, max value: 503</div>					
				</c:if>
				<c:if test="${isElfCell == 1}">
					<span title="<%=rb.getString("RSPCI")%>"><%=rb.getString("PCI2")%></span>
					<input type="text" name="LTE_PHY_CELLID_LIST_1" id="LTE_PHY_CELLID_LIST_1" class="border border-box item" oldValue="${PCI_1}" value="${PCI_1}"
						onblur="validateByRegex(event)" 
						vali-regex="/^(?:(?:(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]),)*(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]))$|^(?:(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3])\.\.(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]))$/"
						title="For example: '23' or '1,2,3' or '300..500' , and every number is between 0 and 503"/>
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
					<div class="errorTitle" id="LTE_PHY_CELLID_LIST_1_err">int, min value: 0, max value: 503</div>					
				</c:if>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSTAC")%>"><%=rb.getString("TAC")%></span>
				<input type="text" name="LTE_TAC_1" id="LTE_TAC_1" class="border border-box item" oldValue="${TAC_1}" value="${TAC_1}"
						onblur="validateMaxAndMinVal1(event)" min_value="0" max_value="65535"
						title="int, min value: 0, max value: 65535"/>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				<div class="errorTitle" id="LTE_TAC_1_err">int, min value: 0, max value: 65535</div>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSReferenceSignalPower")%>"><%=rb.getString("CanKaoXinHaoGongLv")%></span>
				<input type="text" name="LTE_REFERNCE_SIG_POWER_1" id="LTE_REFERNCE_SIG_POWER_1" class="border border-box item" 
					oldValue="${REFERENCE_SIGNAL_POWER_1}" value="${REFERENCE_SIGNAL_POWER_1}"
					onblur="validateMaxAndMinVal1(event)" min_value="-60" max_value="50"
					title="int, min value: -60, max value: 50"/>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				<div class="errorTitle" id="LTE_REFERNCE_SIG_POWER_1_err">int, min value: -60, max value: 50</div>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSBandwidth")%>"><%=rb.getString("DaiKuan")%></span>
				<select name="LTE_UL_BANDWIDTH_1" id="LTE_UL_BANDWIDTH_1" class="border border-box item" >
					<option value=""> </option>
					<!-- <option value="25">5MHz</option> -->
					<option value="50">10MHz</option>
					<!-- <option value="75">15MHz</option> -->
					<option value="100">20MHz</option>
				</select>
				<input type="hidden" value="" id="OLD_LTE_UL_BANDWIDTH_1">
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSSfAssignment")%>"><%=rb.getString("ZiZhenPeiBi")%></span>
				<select name="LTE_TDD_SUBFRAME_ASSIGNMENT_1" id="LTE_TDD_SUBFRAME_ASSIGNMENT_1" class="border border-box item">
					<option value=""> </option>
					<option value="0">SA0</option>
					<option value="1">SA1</option>
					<option value="2">SA2</option>
					<option value="3">SA3</option>
					<option value="4">SA4</option>
					<option value="5">SA5</option>
					<option value="6">SA6</option>
				</select>
				<input type="hidden" value="" id="OLD_LTE_TDD_SUBFRAME_ASSIGNMENT_1">
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSSpecialSfPatterns")%>"><%=rb.getString("TeShuZiZhenPeiBi")%></span>
				<select name="LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1" id="LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1" class="border border-box item">
					<option value=""> </option>
					<option value="0">SSP0</option>
					<option value="1">SSP1</option>
					<option value="2">SSP2</option>
					<option value="3">SSP3</option>
					<option value="4">SSP4</option>
					<option value="5">SSP5</option>
					<option value="6">SSP6</option>
					<option value="7">SSP7</option>
					<option value="8">SSP8</option>
				</select>
				<input type="hidden" value="" id="OLD_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1">
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSRootSequenceIndex")%>"><%=rb.getString("GenXuLieSuoYin")%></span>
				<input type="text" name="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_1"  id="LTE_ROOT_SEQ_INDEX_1" class="border border-box item" 
					oldValue="${ROOT_SEQUENCE_INDEX_1}" value="${ROOT_SEQUENCE_INDEX_1}"
						onblur="validateMaxAndMinVal1(event)" min_value="0" max_value="837"
						title="int, min value: 0, max value: 837"/>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				<div class="errorTitle" id="LTE_ROOT_SEQ_INDEX_1_err">int, min value: 0, max value: 837</div>
			</div>
			<div class="divideLine"></div>
			<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("XiaoQuCanShu")%> 2</span>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSECI")%>"><%=rb.getString("JiZhanID")%></span>
				<input id="LTE_CELL_ECI_2" type="text" name="LTE_CELL_ECI_2" class="border border-box item" oldValue="${CELL_IDENTITY_2}" value="${CELL_IDENTITY_2}"
						onblur="validateMaxAndMinVal1(event)" min_value="0" max_value="268435455" 
						title="int, min value: 0, max value: 268435455"/>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				<div class="errorTitle" id="LTE_CELL_ECI_2_err">int, min value: 0, max value: 268435455</div>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSPLMN")%>"><%=rb.getString("PLMN")%></span>
				<input type="text" name="LTE_OAM_PLMNID_2" id="LTE_OAM_PLMNID_2" class="border border-box item" oldValue="${PLMNID_2}" value="${PLMNID_2}"
						onblur="validateMaxAndMinVal1(event)" min_value="10000" max_value="999999" disabled
						title="<%=rb.getString("PLMNTitle")%>"/>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
				<div class="errorTitle" id="LTE_OAM_PLMNID_2_err"><%=rb.getString("PLMNTitle")%></div>
			</div>
			<div class="itemDiv">
				<c:if test="${isElfCell != 1}">
					<span title="<%=rb.getString("RSPCI")%>"><%=rb.getString("PCI2")%></span>
					<input type="text" name="LTE_PHY_CELLID_LIST_2" id="LTE_PHY_CELLID_LIST_2" class="border border-box item" oldValue="${PCI_2}" value="${PCI_2}"
						onblur="validateMaxAndMinVal1(event)" min_value="0" max_value="503"
						title="int, min value: 0, max value: 503" />
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
					<div class="errorTitle" id="LTE_PHY_CELLID_LIST_2_err">int, min value: 0, max value: 503</div>					
				</c:if>
				<c:if test="${isElfCell == 1}">
					<span title="<%=rb.getString("RSPCI")%>"><%=rb.getString("PCI2")%></span>
					<input type="text" name="LTE_PHY_CELLID_LIST_2" id="LTE_PHY_CELLID_LIST_2" class="border border-box item" oldValue="${PCI_2}" value="${PCI_2}"
						onblur="validateByRegex(event)"
						vali-regex="/^(?:(?:(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]),)*(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]))$|^(?:(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3])\.\.(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]))$/"
						title="For example: '23' or '1,2,3' or '300..500' , and every number is between 0 and 503"/>
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
					<div class="errorTitle" id="LTE_PHY_CELLID_LIST_2_err">int, min value: 0, max value: 503</div>					
				</c:if>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSTAC")%>"><%=rb.getString("TAC")%></span>
				<input type="text" name="LTE_TAC_2"  id="LTE_TAC_2" class="border border-box item" oldValue="${TAC_2}" value="${TAC_2}"
						onblur="validateMaxAndMinVal1(event)" min_value="0" max_value="65535"
						title="int, min value: 0, max value: 65535"/>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				<div class="errorTitle" id="LTE_TAC_2_err">int, min value: 0, max value: 65535</div>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSReferenceSignalPower")%>"><%=rb.getString("CanKaoXinHaoGongLv")%></span>
				<input type="text" name="LTE_REFERNCE_SIG_POWER_2" id="LTE_REFERNCE_SIG_POWER_2" class="border border-box item" 
					oldValue="${REFERENCE_SIGNAL_POWER_2}" value="${REFERENCE_SIGNAL_POWER_2}"
					onblur="validateMaxAndMinVal1(event)" min_value="-60" max_value="50"
					title="int, min value: -60, max value: 50"/>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				<div class="errorTitle" id="LTE_REFERNCE_SIG_POWER_2_err">int, min value: -60, max value: 50</div>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSBandwidth")%>"><%=rb.getString("DaiKuan")%></span>
				<select name="LTE_UL_BANDWIDTH_2" id="LTE_UL_BANDWIDTH_2" class="border border-box item" disabled>
					<option value=""> </option>
					<!-- <option value="25">5MHz</option> -->
					<option value="50">10MHz</option>
					<!-- <option value="75">15MHz</option> -->
					<option value="100">20MHz</option>
				</select>
				<input type="hidden" value="" id="OLD_LTE_UL_BANDWIDTH_2">
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
			</div>
			<div class="itemDiv">

				<span title="<%=rb.getString("RSSfAssignment")%>"><%=rb.getString("ZiZhenPeiBi")%></span>
				<select name="LTE_TDD_SUBFRAME_ASSIGNMENT_2" id="LTE_TDD_SUBFRAME_ASSIGNMENT_2" class="border border-box item" disabled>
					<option value=""> </option>
					<option value="0">SA0</option>
					<option value="1">SA1</option>
					<option value="2">SA2</option>
					<option value="3">SA3</option>
					<option value="4">SA4</option>
					<option value="5">SA5</option>
					<option value="6">SA6</option>
				</select>
				<input type="hidden" value="" id="OLD_LTE_TDD_SUBFRAME_ASSIGNMENT_2">
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
			</div>
			<div class="itemDiv">
				<span title="<%=rb.getString("RSSpecialSfPatterns")%>"><%=rb.getString("TeShuZiZhenPeiBi")%></span>
				<select name="LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_2" id="LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_2" class="border border-box item inputDivCss" disabled>
					<option value=""> </option>
					<option value="0">SSP0</option>
					<option value="1">SSP1</option>
					<option value="2">SSP2</option>
					<option value="3">SSP3</option>
					<option value="4">SSP4</option>
					<option value="5">SSP5</option>
					<option value="6">SSP6</option>
					<option value="7">SSP7</option>
					<option value="8">SSP8</option>
				</select>
				<input type="hidden" value="" id="OLD_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_2">
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
			</div>
			
			<div class="itemDiv">
				<span title="<%=rb.getString("RSRootSequenceIndex")%>"><%=rb.getString("GenXuLieSuoYin")%></span>
				<input type="text" name="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_2"  id="LTE_ROOT_SEQ_INDEX_2" class="border border-box item" 
					oldValue="${ROOT_SEQUENCE_INDEX_2}" value="${ROOT_SEQUENCE_INDEX_2}"
						onblur="validateMaxAndMinVal1(event)" min_value="0" max_value="837"
						title="int, min value: 0, max value: 837"/>
				<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				<div class="errorTitle" id="LTE_ROOT_SEQ_INDEX_2_err">int, min value: 0, max value: 837</div>
			</div>
		</div>
		<%-- 网络设置面板 --%>
		<div class="intentSetting" style="padding: 20px 0 0 20px;">
			<div class="itemDiv">
				<span><%=rb.getString("IPHuoQuFangShi")%></span>
				<select name="UNIT_IP_PROTO"  id="UNIT_IP_PROTO" class="border border-box item" >
					<option value="static">static</option>
					<option value="dhcp">dhcp</option>
			    </select>
			    <input type="hidden" value="" id="OLD_UNIT_IP_PROTO">
			</div>
			<div class="itemDiv staticDiv">
				<span><%=rb.getString("WangGuan")%></span>
				<input type="text" name="UNIT_IP_GATEWAY" id="UNIT_IP_GATEWAY"  class="border border-box staticItem"  oldValue="${GATE_WAY}"
				 		value="${GATE_WAY}" onblur="validateIPAddress1(event)" must="1"
					 	title="<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>"/>
				<div class="errorTitle" id="UNIT_IP_GATEWAY_err"><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></div>
			</div>
			<div class="itemDiv staticDiv">
				<span><%=rb.getString("IPDiZhi")%></span>
				<input type="text" name="UNIT_IP_ADDRESS_NEW" id="UNIT_IP_ADDRESS_NEW" class="border border-box staticItem"  oldValue="${CELL_IP_NEW}"
						value="${CELL_IP_NEW}" onblur="validateIPAddress1(event)" must="1"
					 	title="<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>"/>
				<div class="errorTitle" id="UNIT_IP_ADDRESS_NEW_err"><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></div>
			</div>
			<div class="itemDiv staticDiv">
				<span><%=rb.getString("ZiWangYanMa")%></span>
				<input type="text" name="UNIT_IP_NET_MASK"  id="UNIT_IP_NET_MASK" class="border border-box staticItem"  oldValue="${IP_NET_MASK}"
						value="${IP_NET_MASK}" onblur="validateIPAddress1(event)" must="1"
					 		title="<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>"/>
				<div class="errorTitle" id="UNIT_IP_NET_MASK_err"><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></div>
			</div>
			<div class="itemDiv staticDiv" style="display: none">
				<span><%=rb.getString("MoRenDnsDiZhi1")%></span>
				<input type="text" disabled name="UNIT_IP_DNS1"  id="UNIT_IP_DNS1" class="border border-box staticItem"  value="114.114.114.114"/>
			</div>
			<div class="itemDiv staticDiv">
				<span><%=rb.getString("MoRenDNS")%></span>
				<input type="text" disabled name="UNIT_IP_DNS2"  id="UNIT_IP_DNS2"class="border border-box staticItem"  value="8.8.8.8"/>
			</div>
			<div class="itemDiv staticDiv">
				<span><%=rb.getString("ZiDingYiDnsDiZhi")%></span>
				<input type="text" name="UNIT_IP_DNS"  id="UNIT_IP_DNS" class="border border-box staticItem" oldValue=""  
						onblur="validateIPAddress1(event)"
					 	title="<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>"/>
				<div class="errorTitle" id="UNIT_IP_DNS_err"><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></div>
			</div>
		</div>
		<!-- NTP设置 -->
		<div class="NTPSetting" id="compactParamConfigTabs" style="padding: 20px 0 0 20px;">
			<div class="itemDiv" style="float:none;clear:both;">
				<span><%=rb.getString("NTPTongBuShiJian")%></span>
				<input type="text" name="LTE_X_BAICELLS_NTP_SYNC_INTERVAL" id="LTE_X_BAICELLS_NTP_SYNC_INTERVAL" class="border border-box item" oldValue="${X_BAICELLS_NTP_SYNC_INTERVAL}" value="${X_BAICELLS_NTP_SYNC_INTERVAL}"
						onblur="validateMaxAndMinVal1(event)" min_value="10" max_value="65535"
						title="int, min value: 10, max value: 65535"/>
				<div class="errorTitle" id="LTE_X_BAICELLS_NTP_SYNC_INTERVAL_err">int, min value: 10, max value: 65535</div>
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("NTPFuWuQi1") %></span>
				<input type="text" name="LTE_NTP_SERVER_1"  id="LTE_NTP_SERVER_1" class="border border-box item" oldValue="${NTP_SERVER_1}" value="${NTP_SERVER_1}"
						onblur="validateMaxAndMinLength1(event)" min_length="0" max_length="256"
						title="string, min length 0, max length 256"/>
				<div class="errorTitle" id="LTE_NTP_SERVER_1_err">String,min length 0,max length 256</div>
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("NTPDuanKou1") %></span>
				<input type="text" name="LTE_NTP_PORT_1"  id="LTE_NTP_PORT_1" class="border border-box item" oldValue="${NTP_PORT_1}" value="${NTP_PORT_1}"
						onblur="validateMaxAndMinVal1(event)" min_value="1" max_value="20000"
						title="int, min value: 1, max value: 20000"/>
				<div class="errorTitle" id="LTE_NTP_PORT_1_err">int, min value: 1, max value: 20000</div>
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("NTPFuWuQi2") %></span>
				<input type="text" name="LTE_NTP_SERVER_2" id="LTE_NTP_SERVER_2"  class="border border-box item" oldValue="${NTP_SERVER_2}" value="${NTP_SERVER_2}"
						onblur="validateMaxAndMinLength1(event)" min_length="0" max_length="256"
						title="string, min length 0, max length 268435455"/>
				<div class="errorTitle" id="LTE_NTP_SERVER_2_err">String,min length 0,max length 256</div>
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("NTPDuanKou2") %></span>
				<input type="text" name="LTE_NTP_PORT_2"  id="LTE_NTP_PORT_2" class="border border-box item" oldValue="${NTP_PORT_2}" value="${NTP_PORT_2}"
						onblur="validateMaxAndMinVal1(event)" min_value="1" max_value="20000"
						title="int, min value: 1, max value: 20000"/>
				<div class="errorTitle" id="LTE_NTP_PORT_2_err">int, min value: 1, max value: 20000</div>
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("NTPFuWuQi3") %></span>
				<input type="text" name="LTE_NTP_SERVER_3" id = "LTE_NTP_SERVER_3" class="border border-box item" oldValue="${NTP_SERVER_3}" value="${NTP_SERVER_3}"
						onblur="validateMaxAndMinLength1(event)" min_length="0" max_length="256"
						title="string, min length 0, max length 256"/>
				<div class="errorTitle" id="LTE_NTP_SERVER_3_err">String,min length 0,max length 256</div>
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("NTPDuanKou3") %></span>
				<input type="text" name="LTE_NTP_PORT_3"  id="LTE_NTP_PORT_3" class="border border-box item" oldValue="${NTP_PORT_3}" value="${NTP_PORT_3}"
						onblur="validateMaxAndMinVal1(event)" min_value="1" max_value="20000"
						title="int, min value: 1, max value: 20000"/>
				<div class="errorTitle" id="LTE_NTP_PORT_3_err">int, min value: 1, max value: 20000</div>
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("NTPFuWuQi4") %></span>
				<input type="text" name="LTE_NTP_SERVER_4"  id="LTE_NTP_SERVER_4" class="border border-box item" oldValue="${NTP_SERVER_4}" value="${NTP_SERVER_4}"
						onblur="validateMaxAndMinLength1(event)" min_length="0" max_length="256"
						title="string, min length 0, max length: 256"/>
				<div class="errorTitle" id="LTE_NTP_SERVER_4_err">String,min length 0,max length 256</div>
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("NTPDuanKou4") %></span>
				<input type="text" name="LTE_NTP_PORT_4" id="LTE_NTP_PORT_4"  class="border border-box item" oldValue="${NTP_PORT_4}" value="${NTP_PORT_4}"
						onblur="validateMaxAndMinVal1(event)" min_value="1" max_value="20000"
						title="int, min value: 1, max value: 20000"/>
				<div class="errorTitle" id="LTE_NTP_PORT_4_err">int, min value: 1, max value: 20000</div>
			</div>
		</div>
		<c:if test="${lgw == 1}">
			<div class="LGWSetting" style="padding: 20px 0 0 20px;" id="lgw">
				<div class="itemDiv">
					<span><%=rb.getString("LGWKaiGuan")%></span>
					<select name="LGW_ENABLE"  id="intelTdd_LGW_ENABLE" class="border border-box lgwInput" onchange="lgwEnableChange(this)">
						<option value="1"><%=rb.getString("LGWKaiQi")%></option>
						<option value="0"><%=rb.getString("LGWGuanBi")%></option>
				    </select>
				    <input type="hidden" value="${LGW_SWITCH}" id="intelTdd_OLD_LGW_ENABLE">
				</div>
				
				<div class="itemDiv" name="LGW">
					<span><%=rb.getString("LGWMoShi")%></span>
					<select name="LGW_MODE"  id="intelTdd_LGW_MODE" class="border border-box lgwInput" onchange="lgwModeChange(this)">
						<option value="NAT" selected>NAT</option>
						<option value="Router">Router</option>
						<option value="Bridge">Bridge</option>
				    </select>
				    <input type="hidden" value="${LGW_TRANSFER_MODE}" id="intelTdd_OLD_LGW_MODE">
				    <div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>		
				</div>
								
				<div class="itemDiv" name="LGW">
					<span><%=rb.getString("LGWJieKouBangDing")%></span>
					<select name="LGW_INTERFACE_BINDING"  id="intelTdd_LGW_INTERFACE_BINDING" class="border border-box lgwInput" disabled >
						<option value="wan" seleted>WAN</option>
						<!-- <option value="pppoe">PPPOE</option> -->
						<!-- <option value="vlan 1">VLAN 1</option> -->
				    </select>
				    <input type="hidden" value="${LGW_IFNAME}" id="intelTdd_OLD_LGW_INTERFACE_BINDING">
				</div>
				
				<div class="itemDiv" name="LGW">
					<span><%=rb.getString("LGWDiZhiChi")%></span>
					<input type="text" name="LGW_IP_POOL"  id="intelTdd_LGW_IP_POOL" class="border border-box lgwInput"  oldValue="${LGW_START_UE_ADDRD}" value="10.0.0.1" 
							onblur="ippoolonblur(event)" must="1" title="<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>"/>
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
					<div class="errorTitle" id="intelTdd_LGW_IP_POOL_err"><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></div>	
				</div>
				
				<div class="itemDiv" name="LGW">
					<span><%=rb.getString("LGWDiZhiChiYanMa")%></span>
					<select name="LGW_IP_POOL_NETMASK"  id="intelTdd_LGW_IP_POOL_NETMASK" class="border border-box lgwInput"  >
						<option value="255.255.255.0" seleted>255.255.255.0</option>
						<option value="255.255.255.128" >255.255.255.128</option>
						<option value="255.255.255.192" >255.255.255.192</option>
						<option value="255.255.255.224" >255.255.255.224</option>
						<option value="255.255.255.240" >255.255.255.240</option>
						<option value="255.255.255.248" >255.255.255.248</option>
				    </select>
				    <input type="hidden" value="${LGW_NET_MASK}" id="intelTdd_OLD_LGW_IP_POOL_NETMASK">
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
				</div>
				

				<div class="itemDiv staticDiv" name="LGW">
					<span><%=rb.getString("JingTaiDiZhi")%></span>
					<select name="LGW_STATIC_ADDRESS"  id="intelTdd_LGW_STATIC_ADDRESS" class="border border-box lgwInput" onchange="staticIPEnable(this)">
						<option value="1"><%=rb.getString("LGWKaiQi")%></option>
						<option value="0" selected><%=rb.getString("LGWGuanBi")%></option>
				    </select>
				    <input type="hidden" value="${LGW_STATIC_IP_ADDR_SWITCH}" id="intelTdd_OLD_LGW_STATIC_ADDRESS">
				</div>
				
				<div class="itemDiv staticDiv" name="LGW">
					<span style="display:block"><%=rb.getString("JingTaiDiZhiFanWeiPeiZhi")%></span>
					<input type="text" name="LGW_STATIC_IP_CONFIG_BEGIN" id="intelTdd_LGW_STATIC_IP_CONFIG_BEGIN" class="border border-box itemStatisticInput" style="width:169px;height:26px;padding-left:10px"  oldValue="${LGW_FIRST_STATIC_IP_ADDRESS}"
					 onblur="validateipandrange(event)" must="1"/>
					-
					<input type="text" name="LGW_STATIC_IP_CONFIG_END" id="intelTdd_LGW_STATIC_IP_CONFIG_END" class="border border-box itemStatisticInput" style="width:169px;height:26px;padding-left:10px"  oldValue="${LGW_LAST_STATIC_IP_ADDRESS}"
					 onblur="validateipandrange(event)" must="1"/>
					<div class="promptTitle" id="lgwiprange" name="LGW" style="color:#4c6778"><%=rb.getString("IPBangDingFanWei")%><span style="margin-left:10px;">10.0.0.1-10.0.0.100</span></div>
				    <div class="errorTitle" id="intelTdd_LGW_STATIC_IP_CONFIG_err"><%=rb.getString("IPBangDingFanWei")%><span style="margin-left:10px;">10.0.0.1-10.0.0.100</span></div>
				</div>
				
				
				<div class="itemDiv staticDiv lgw" name="LGW" id="1" style="float:none;clear:both;">
					<span style="display:block"><%=rb.getString("IMSIBangDing")%></span>
					<input type="text"  name="LGW_IMSI"  id="intelTdd_LGW_IMSI" class="border border-box itemStatisticInput" style="width:169px;height:26px;padding-left:10px" oldValue="${LGW_IMSI}" onblur="imsionblur(event)" title="<%=rb.getString("LGWImsiTiShi")%>"/>
					-
					<input type="text"  name="LGW_IMSI_IP"  id="intelTdd_LGW_IMSI_IP"  class="border border-box itemStatisticInput" style="width:169px;height:26px;padding-left:10px" onblur="validateipandrangeforbinding(event)" oldValue="${LGW_IMSI_IP}" title="<%=rb.getString("LGWIpTiShi")%>"/>
					<a onclick="intelTdd_addISMIAndIPInputText(this)">
						<img src="${ctx}/skin/${manufacturer}/images/bi/setting_add.png"/>  
					</a>
				    <div class="errorTitle" style="height:21px;line-height:21px;" id="intelTdd_LGW_IMSI_err"><%=rb.getString("LGWImsiTiShi")%></span></div>
				    <div class="errorTitle" style="height:21px;line-height:21px;" id="intelTdd_LGW_IMSI_IP_err"><%=rb.getString("LGWIpTiShi")%></span></div>
				</div>
				
				<div id = "after_log_ismi_ip"></div>
				
			</div>
			</c:if>
	</div>
	<div class="windowButtonGroup" style="position:absolute;bottom:37px;left:47px;">
		<a class="easyui-linkbutton linkbutton linkbutton_trend" onclick="cellSettingCommit()"><%=rb.getString("QueDing")%></a>
		<a class="easyui-linkbutton linkbutton linkbutton_nowanna" onclick="cellSettingCancel()"><%=rb.getString("QuXiao")%></a>
	</div>
	</div>
</div>

<%-- 窗口-右键设置进度条 --%>
<div id="winSettingPro" title="<%=rb.getString("CanShuPeiZhiJinDu")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:400,height:70,resizable:false,closable:false">
    <img src="${ctx}/js/jquery-easyui/themes/bootstrap/images/loading.gif"/>
    <span style="margin-left:75px;"><%=rb.getString("eNodeBZhengZaiSheZhi")%></span>
   
</div>

<script type="text/javascript">
$("select[name='LTE_UL_BANDWIDTH_1']").bind("change",function(){
	var ele = $(this);
	//移除之前被选中的元素
	$("select[name='LTE_DL_BANDWIDTH'] option:selected").removeAttr("selected");
	//选中新选择的元素
	var opts = $("select[name='LTE_UL_BANDWIDTH_1']").find("option");
	var oldValue=$("#OLD_LTE_UL_BANDWIDTH_1").val();
	var oldValueText = "";
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == oldValue){
			oldValueText = $(opts[i]).text();
		}
	}
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == this.value){
			if(oldValue!=this.value){
				$("#LTE_UL_BANDWIDTH_1").css({"color":"blue"});
				$("#LTE_UL_BANDWIDTH_1 option").css({"color":"black"});
				ele.siblings(".promptTitle").show();
		    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValueText);
			}else{
				$("#LTE_UL_BANDWIDTH_1").css({"color":"black"});	
				ele.siblings(".promptTitle").hide();	
			}
			break;
		}
	}
	
	
});

$("select[name='LTE_TDD_SUBFRAME_ASSIGNMENT_1']").bind("change",function(){
	//选中新选择的元素
	var opts = $("select[name='LTE_TDD_SUBFRAME_ASSIGNMENT_1']").find("option");
	var oldValue=$("#OLD_LTE_TDD_SUBFRAME_ASSIGNMENT_1").val();
	var ele = $(this);
	var oldValueText = "";
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == oldValue){
			oldValueText = $(opts[i]).text();
		}
	}
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == this.value){
			$(opts[i]).attr("selected",true);
			if(oldValue!=this.value){
				$("#LTE_TDD_SUBFRAME_ASSIGNMENT_1").css({"color":"blue"});
				$("#LTE_TDD_SUBFRAME_ASSIGNMENT_1 option").css({"color":"black"});
				ele.siblings(".promptTitle").show();
		    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValueText);	
			}else{
				$("#LTE_TDD_SUBFRAME_ASSIGNMENT_1").css({"color":"black"});	
				ele.siblings(".promptTitle").hide();	
			}
			break;
		}
	}
	
});

$("select[name='LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1']").bind("change",function(){
	//选中新选择的元素
	var opts = $("select[name='LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1']").find("option");
	var oldValue=$("#OLD_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1").val();
	var ele = $(this);
	var oldValueText = "";
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == oldValue){
			oldValueText = $(opts[i]).text();
		}
	}
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == this.value){
			$(opts[i]).attr("selected",true);
			if(oldValue!=this.value){
				$("#LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1").css({"color":"blue"});
				$("#LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1 option").css({"color":"black"});
				ele.siblings(".promptTitle").show();
		    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValueText);	
			}else{
				$("#LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1").css({"color":"black"});	
				ele.siblings(".promptTitle").hide();	
			}
			break;
		}
	}
});

$("select[name='UNIT_IP_PROTO']").bind("change",function(){
	//选中新选择的元素
	var opts = $("select[name='UNIT_IP_PROTO']").find("option");
	var oldValue=$("#OLD_UNIT_IP_PROTO").val();
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == this.value){
			$(opts[i]).attr("selected",true);
			if(oldValue!=this.value){
				$("#UNIT_IP_PROTO").css({"color":"blue"});
				$("#UNIT_IP_PROTO option").css({"color":"black"});
			}else{
				$("#UNIT_IP_PROTO").css({"color":"black"});	
			}
			break;
		}
	}
	
});


$("select[id='AK_cert']").bind("change",function(){
	//选中新选择的元素
	var opts = $("select[id='AK_cert']").find("option");
	var oldValue=$("#old_AK_cert").val();
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == this.value){
			$(opts[i]).attr("selected",true);
			if(oldValue!=this.value){
				$("#AK_cert").css({"color":"blue"});
				$("#AK_cert option").css({"color":"black"});
			}else{
				$("#AK_cert").css({"color":"black"});	
			}
			break;
		}
	}
	
});


function authbyChange(e){
	var ele = $(e["target"]);
	var name = ele.attr("name");
	var value = ele.val();
	var oldValue = ele.nextAll("input").eq(0).val();
	var type = /^[^\d]*(\d+)$/;
	var index = type.exec(name)[1];
	if(oldValue == "cert" || oldValue == "AKA_cert"){
		/* if(value == "psk" || value == "AKA_psk"){
			$("input[name='ROOTCERTIFICATES_"+index+"']").parent().hide();
			$("input[name='CERTIFICATES_"+index+"']").parent().hide();
			$("input[name='PRIVATEKEYS_"+index+"']").parent().hide();
		}else{
			$("input[name='ROOTCERTIFICATES_"+index+"']").parent().show();
			$("input[name='CERTIFICATES_"+index+"']").parent().show();
			$("input[name='PRIVATEKEYS_"+index+"']").parent().show();
		} */
		if(oldValue == "cert" && value == "AKA_cert"){
			$.messager.alert(TiShi, "<%=rb.getString("BuNnengGengGai")%>");
			ele.val(oldValue);
		}else if(oldValue == "AKA_cert" && value == "cert"){
			$.messager.alert(TiShi, "<%=rb.getString("BuNnengGengGai")%>");
			ele.val(oldValue);
		}
	}else{
		if(value == "cert" || value == "AKA_cert"){
			$.messager.alert(TiShi, "<%=rb.getString("BuNnengGengGai")%>");
			ele.val(oldValue);
		}
	}
	chageColor();
}
<%-- 加载完成事件 --%>
$(function() {
	
	closeLoading();
	
	/* 判断是否为移动版本，移动版本下不显示锁定频点、PCI标识 */
	if(omcVersion == 1) {
		$(".pciisLock").find("img").hide();
	} else {
		$(".pciisLock").find("img").show();
	}
	/*  判断CPE下的PCI是否锁定 */
	//如果此基站有CPE进行锁频，那么对应的pci，bandclass,earfcn3个输入框将变为不可编辑
	var PCIIsLock ="${hasCpeFreqLock}";
	if(!PCIIsLock){
		$('.pciisLock img').attr("src","${ctx}/skin/${manufacturer}/images/bi/pci_unlock.png");
	}else{
		$('.pciisLock img').attr("src","${ctx}/skin/${manufacturer}/images/bi/pci_lock.png");
		$('.pciisLock img').attr('title','This device is locked')
		$('.pciisLock input').attr('disabled','disabled')
	}
	
	//如果SAS开关打开，则该参数不可配置
	if(SASEnble == "1"){
		$("#LTE_UL_BANDWIDTH_1").attr("disabled",true).css("background","#EAF1F4");
		$("#LTE_UL_BANDWIDTH_2").attr("disabled",true).css("background","#EAF1F4");
	}
	
	/* var isCpeFreqLock = "${hasCpeFreqLock}";
	//如果此基站有CPE进行锁频，那么对应的pci，bandclass,earfcn3个输入框将变为不可编辑
	if (isCpeFreqLock == 1) {
		$("input[name='LTE_PHY_CELLID_LIST']").attr("disabled", true);
		//$("input[name='LTE_FREQ_BAND_INDICATOR']").attr("disabled", true);
		$("input[name='LTE_UL_DL_EARFCN']").attr("disabled", true);
	} */
	
	<%-- 为下拉框赋值 --%>
	$("select[name='LTE_DL_BANDWIDTH']").val("${DL_BANDWIDTH}");
	$("select[name='LTE_UL_BANDWIDTH_1']").val("${UL_BANDWIDTH_1}");
	$("select[name='LTE_UL_BANDWIDTH_2']").val("${UL_BANDWIDTH_2}");
	$("select[name='LTE_TDD_SUBFRAME_ASSIGNMENT_1']").val("${SUBFRAME_ASSIGNMENT_1}");
	$("select[name='LTE_TDD_SUBFRAME_ASSIGNMENT_2']").val("${SUBFRAME_ASSIGNMENT_2}");
	$("select[name='LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1']").val("${SPECIAL_SUBFRAME_PATTERNS_1}");
	$("select[name='LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_2']").val("${SPECIAL_SUBFRAME_PATTERNS_2}");
	//值为空表示不支持IP_PROTO参数
	if ("${IP_PROTO}" == "") {
		$("select[name='UNIT_IP_PROTO']").val("dhcp");
		$("select[name='UNIT_IP_PROTO']").attr("disabled","disabled");
		$("#OLD_UNIT_IP_PROTO").val("dhcp");
	} else {
		$("select[name='UNIT_IP_PROTO']").val("${IP_PROTO}");
		$("#OLD_UNIT_IP_PROTO").val("${IP_PROTO}");
	}
	
	$("#OLD_LTE_DL_BANDWIDTH").val("${DL_BANDWIDTH}");
	$("#OLD_LTE_UL_BANDWIDTH_1").val("${UL_BANDWIDTH_1}");
	$("#OLD_LTE_UL_BANDWIDTH_2").val("${UL_BANDWIDTH_2}");
	$("#OLD_LTE_TDD_SUBFRAME_ASSIGNMENT_1").val("${SUBFRAME_ASSIGNMENT_1}");
	$("#OLD_LTE_TDD_SUBFRAME_ASSIGNMENT_2").val("${SUBFRAME_ASSIGNMENT_2}");
	$("#OLD_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_1").val("${SPECIAL_SUBFRAME_PATTERNS_1}");
	$("#OLD_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_2").val("${SPECIAL_SUBFRAME_PATTERNS_2}");

	
	var isSuperAdmin = $("#isSuperAdmin").val();
	<%-- 生成Ipsec--%>
	var cellIpsecInfos = ${cellIpsecInfos};
	if(cellIpsecInfos.length==0){
		$("#IPSECDIV").append('<%=rb.getString("IpsecTiShi") %>');
	}else{
		for(var i=0;i<cellIpsecInfos.length;i++){
			var ID = cellIpsecInfos[i].ID;
			//创建ID文本框
			var IDDIV = createIpsecInputDiv('<%=rb.getString("SuoYin") %>',"ID_"+ID,ID,true);
			var TunnelMingChengDIV = createIpsecInputDiv('<%=rb.getString("TunnelMingCheng") %>',"TUNNEL_NAME_"+ID,cellIpsecInfos[i].TUNNEL_NAME);
			var TunnelWangGuanDIV = createIpsecInputDiv('<%=rb.getString("TunnelWangGuan") %>',"TUNNEL_GATEWAY_"+ID,cellIpsecInfos[i].TUNNEL_GATEWAY);
			var IpsecZiWangDIV = createIpsecInputDiv('<%=rb.getString("IpsecZiWang") %> ',"RIGHT_SUBNET_"+ID,cellIpsecInfos[i].RIGHT_SUBNET);
			var MuDiDuanBiaoJiFuDIV = createIpsecInputDiv('<%=rb.getString("MuDiDuanBiaoJiFu") %>',"RIGHT_IDENTIFIER_"+ID,cellIpsecInfos[i].RIGHT_IDENTIFIER);
			var RenZhengFangShiDIV =createIpsecSelectDiv({"psk":"psk","cert":"cert","AKA_psk":"AKA_psk","AKA_cert":"AKA_cert"},cellIpsecInfos[i].AUTHBY,'<%=rb.getString("RenZhengFangShi") %>',"AUTHBY_"+ID);
			var YuGongXiangMiYaoDIV = createIpsecInputPasswordDiv('<%=rb.getString("YuGongXiangMiYao") %>',"PRE_SHARED_KEY_"+ID,cellIpsecInfos[i].PRE_SHARED_KEY);
			//OMC不支持cert认证方式，如果需要使用cert认证方式，则需要在小站上进行配置。假设之小站配置为psk或者AKA_psk,则修改认证方式为cert或者AKA_cert时给出提示信息，不允许修改
			//小站上配置认证方式为cert或者AKA_cert，则OMC要显示认证方式为cert或者AKA_cert，认证方式可以做修改，但是具体要不要显示其对应的三个证书待定，故先注释掉，
			//if(cellIpsecInfos[i].AUTHBY == "cert" || cellIpsecInfos[i].AUTHBY == "AKA_cert"){
				//var RootCertificatesDIV = createIpsecInputDiv('<%=rb.getString("GenZhengShu") %>',"ROOTCERTIFICATES_"+ID,cellIpsecInfos[i].ROOTCERTIFICATES,true);
				//var CertificatesDIV = createIpsecInputDiv('<%=rb.getString("ZhengShu") %>',"CERTIFICATES_"+ID,cellIpsecInfos[i].CERTIFICATES,true);
				
				
				//var PrivateKeysDIV = createIpsecInputDiv('<%=rb.getString("SiYaoZhengShu") %>',"PRIVATEKEYS_"+ID,cellIpsecInfos[i].PRIVATEKEYS,true);
				//$("#IPSECDIV").append(IDDIV).append(TunnelMingChengDIV).append(TunnelWangGuanDIV).append(IpsecZiWangDIV).append(MuDiDuanBiaoJiFuDIV).append(RenZhengFangShiDIV).
				//append(RootCertificatesDIV).append(CertificatesDIV).append(PrivateKeysDIV).append(YuGongXiangMiYaoDIV);	
			//}else{
				//$("#IPSECDIV").append(IDDIV).append(TunnelMingChengDIV).append(TunnelWangGuanDIV).append(IpsecZiWangDIV).append(MuDiDuanBiaoJiFuDIV).append(RenZhengFangShiDIV).append(YuGongXiangMiYaoDIV);	
			//}
			if(isSuperAdmin == 1){//超级管理员可以查看预共享秘钥
				$("#IPSECDIV").append(IDDIV).append(TunnelMingChengDIV).append(TunnelWangGuanDIV).append(IpsecZiWangDIV).append(MuDiDuanBiaoJiFuDIV).append(RenZhengFangShiDIV).append(YuGongXiangMiYaoDIV);
			}else{
				$("#IPSECDIV").append(IDDIV).append(TunnelMingChengDIV).append(TunnelWangGuanDIV).append(IpsecZiWangDIV).append(MuDiDuanBiaoJiFuDIV).append(RenZhengFangShiDIV);
			}
			
		}
		
		$("input[name^='TUNNEL_NAME']").blur(validateMaxAndMinLength).attr("min_length",1).attr("max_length",14).attr("title","string, min length: 1, max length: 14");
		$("input[name^='TUNNEL_GATEWAY']").blur(validateMaxAndMinLength).attr("min_length",7).attr("max_length",15).attr("title","string, min length: 7, max length: 15");
		$("input[name^='RIGHT_SUBNET']").blur(validateMaxAndMinLength).attr("min_length",0).attr("max_length",500).attr("title","string, min length: 0, max length: 500");
		$("input[name^='RIGHT_IDENTIFIER']").blur(validateMaxAndMinLength).attr("min_length",0).attr("max_length",500).attr("title","string, min length: 0, max length: 500");
		$("input[name^='PRE_SHARED_KEY_']").blur(validateMaxAndMinLength).attr("min_length",0).attr("max_length",500).attr("title","string, min length: 0, max length: 500");
		
		$("select[name^='AUTHBY']").change(authbyChange);
		
	}
	
	//如果是dhcp协议获取ip地址，ip地址，网关和子网掩码隐藏
	var unit_ip_proto = $("select[name='UNIT_IP_PROTO']").val();
	if( unit_ip_proto == "dhcp"){
		$(".staticDiv").each(function() {
			$(this).css({
				"display" : "none"
			});
		});
	}
	
	//监控用户选择ip方式的change事件
	$("select[name='UNIT_IP_PROTO']").change(function(){
		//获取用户选择的方式，如果是dhcp，则隐藏ip，网关，子网掩码
		var selectProto = $("select[name='UNIT_IP_PROTO']").val();
		if(selectProto == "dhcp") {
			$(".staticDiv").each(function() {
				$(this).css({
						"display" : "none"
				});
			});
		}else {
			$(".staticDiv").each(function() {
				$(this).css({
						"display" : "block"
				});
			});
			$("input[name='UNIT_IP_DNS1']").parent().css({"display":"none"});
		}
	});
	
	//初始化自定义dns字段
	var ipDns = "${IP_DNS}";
	if (ipDns.length > 0) {
		var ipDnsArr = ipDns.split(",");
		var ipDnsLength = ipDnsArr.length;
		for (var count = 0; count < ipDnsLength; count++) {
			var dnsStr = ipDnsArr[count];
			if (dnsStr.length > 0) {
				if (dnsStr != "114.114.114.114" && dnsStr != "8.8.8.8") {
					$("input[name='UNIT_IP_DNS']").val(dnsStr);
					$("input[name='UNIT_IP_DNS']").attr("oldValue", dnsStr);
				}
			}
		}
	}
	
	//初始化核心网地址输入框
	var mmeAddress = "${S1SIGLINKSERVERLIST}";
	if (mmeAddress != "") {
		var mmeArr = mmeAddress.split(",");
		//当mme地址为n且n>1时，需要重新生成n-1个输入框
		if (mmeArr.length > 1) {
			//初始化第一个输入框的值和oldValue值
			var firstMMEDiv = $("input[name='LTE_SIGLINK_SERVER_LIST']:first");
			firstMMEDiv.val(mmeArr[0]);
			firstMMEDiv.attr("oldValue", mmeArr[0]);
			
			var newMMEInputCount = mmeArr.length;
			//生成n-1个输入框
			for(var count = 1; count < newMMEInputCount; count++) {
				//添加新的输入框
				var $mmeSpan = $("<span style='display:inline-block;visibility:hidden'><%=rb.getString("HeXinWang")%></span>");
				var $mmeDiv = $("<div class='itemDiv'></div>");
				var $mmeInput = $("<input type='text' name='LTE_SIGLINK_SERVER_LIST' class='border border-box item' "
					 + "onblur='validateIPAddress(event)' oldValue='" + mmeArr[count] + "' value='" + mmeArr[count] + "' style='margin-left:4px'></input>");
				var $mmeButtn = $("<a onclick='removeMMEIpInputText(this)' style='margin-left:3px'>" 
						+ "<img src='${ctx}/skin/${manufacturer}/images/bi/setting_sub.png'/> </a>");
				
				$mmeDiv.append($mmeSpan);
				$mmeDiv.append($mmeInput);
				$mmeDiv.append($mmeButtn);
				
				//将新增的输入框插入到plmn参数之前
				var plmnidDiv = $("input[name='LTE_OAM_PLMNID']").parent("div");
				$mmeDiv.insertBefore(plmnidDiv);
			}
		} else {
			var firstMMEDiv = $("input[name='LTE_SIGLINK_SERVER_LIST']:first");
			firstMMEDiv.val(mmeArr[0]);
			firstMMEDiv.attr("oldValue", mmeArr[0]);
		}
	}
	initlgw();
});
$("#intelTdd_LGW_IP_POOL_NETMASK").bind("change",function(){
	//移除之前被选中的元素
	//$("#intelTdd_LGW_IP_POOL_NETMASK option:selected").removeAttr("selected");
	//选中新选择的元素
	var opts = $("#intelTdd_LGW_IP_POOL_NETMASK").find("option");
	var oldValue=$("#intelTdd_OLD_LGW_IP_POOL_NETMASK").attr("value");
	var ele = $(this);
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == this.value){
			$(opts[i]).attr("selected",true);
			if(oldValue!=this.value){
				$("#intelTdd_LGW_IP_POOL_NETMASK").css({"color":"blue"});
				$("#intelTdd_LGW_IP_POOL_NETMASK option").css({"color":"black"});
				ele.siblings(".promptTitle").show();
		    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValue);
			}else{
				$("#intelTdd_LGW_IP_POOL_NETMASK").css({"color":"black"});	
				ele.siblings(".promptTitle").hide();
			}
			break;
		}
	}
	
	var ipvalue = $("#intelTdd_LGW_IP_POOL").val();
	var ippoolmask = $("#intelTdd_LGW_IP_POOL_NETMASK").val();
	var lowip = getLowAddr(ipvalue,ippoolmask);
	var highip = getHighAddr(ipvalue,ippoolmask);
	$("#lgwiprange span").html(lowip + "-" + highip);
	$("#intelTdd_LGW_STATIC_IP_CONFIG_err span").html(lowip + "-" + highip);
	if($("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").val() != ''){
		$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").blur();
	}
	if($("#intelTdd_LGW_STATIC_IP_CONFIG_END").val() != ''){
		$("#intelTdd_LGW_STATIC_IP_CONFIG_END").blur();
	}
	var IMCIIPinput = $("input[name=LGW_IMSI_IP]");
	$(IMCIIPinput).each(function(index,ele){
		if($(ele).val() != ''){
			$(ele).blur();
		}
	});
});
<%-- 提交 --%>
function cellSettingCommit() {
	var cellCode = "<%=request.getParameter("smallCellCode") %>";
	var lgwchange = false;
	var icicreboot = false;
	//判断基站是否连接
	$.post("${ctx}/cell/cpeinfos/getCellConnStatus.action", {"small_cell_code": cellCode}, function(data){
		// 基站未连接，不允许修改参数
		if(data["connStatus"] == "false") {
			$.messager.alert(TiShi, "<%=rb.getString("JiZhanWeiLianJie")%>");
			return;
		} else {
			if ($("#compactParamConfig .item.err_border").length > 0) {
				return;
			}
			if($("select[name='LGW_ENABLE']").val()=="1"){
				var lgwModeSelect = $("select[name='LGW_MODE']").val();
				if(lgwModeSelect=='NAT'){
					if ($("#compactParamConfig .lgwInput.err_border").length > 0) {
						return;
					}
				}else if(lgwModeSelect=='Router'){					
					if ($("#compactParamConfig .lgwInput.err_border").length > 0) {
						return;
					}
					var selectStaticAddrInput = $("select[name='LGW_STATIC_ADDRESS']").val();
					if(selectStaticAddrInput == "1") {
						if ($("#compactParamConfig .itemStatisticInput.err_border").length > 0) {
							return;
						}
					}
				}
			}
			var selectProto = $("select[name='UNIT_IP_PROTO']").val();
			if(selectProto == "static") {
				if ($("#compactParamConfig .staticItem.err_border").length > 0) {
					return;
				}
			}
			var paramsForeNodeBName = {};
			var params = {};
			//是否修改ip获取方式相关的参数
			var isChangeIpProto = false;
			
			$("#compactParamConfig .item,.lgwInput,.itemStatisticInput").each(function() {
				var paramName = $(this).attr("name")||$(this).attr("textboxname");
				var newValue = $(this).val();
				var oldValue = $(this).attr("oldValue");
				if(oldValue == undefined){
					//下拉框得参数为获取id为OLD_+name的value值
					var selectId = "OLD_" + paramName;
					var reg = new RegExp("^LGW");
					if(reg.test(paramName)){
						oldValue = $("#intelTdd_" + selectId).val();
					} else {
						oldValue = $("#" + selectId).val();
					}
				}
				//核心网地址参数特殊处理
				if (paramName == "LTE_SIGLINK_SERVER_LIST") {
					//如果是核心网地址且不为空，则将多个参数值拼接起来
					if (params[paramName] != null) {
						var mmeAdd = params[paramName];
						//确保不添加重复地址
						if (mmeAdd.indexOf(newValue) == -1) {
							params[paramName] =  params[paramName] + newValue + ","; 
						}
					} else {
						params[paramName] =  newValue + ","; 
					}
				/* } else if (paramName == "LTE_HOME_NODEB_NAME") {//配置基站名称需要特殊处理
					if (oldValue != newValue) {
						var paramsForCellName = {
								eNodeBName : newValue,
								small_cell_code : cellCode
						};
						$.post("${ctx}/cell/cpeinfos/updateCellName.action", paramsForCellName, function(data){
							$("#tableHomeCellList").datagrid("reload");
							if (!data["success"]) {
								$.messager.alert(TiShi, data["message"]);
							}
						}, "json");
						paramsForeNodeBName["changed"] = true;
					} */
				} else {
					//其他的参数值变化才加入到params中
					if (oldValue != newValue) {
						var reg = new RegExp("^LGW");
						if(!reg.test(paramName)){
							if (paramName == "LTE_UL_DL_EARFCN") {
								if ("1" == "${isElfCell}") {// elfCell网管，特殊处理
									params["LTE_EARFCN_ELFCELL"] = newValue;
								} else {// 普通基站网管
									//上行频点和下行频点合并为频点，因此需要进行特殊处理
									params["LTE_DL_EARFCN"] = newValue;
									params["LTE_UL_EARFCN"] = newValue;
								}
							} else {
								params[paramName] =  newValue;
							}
							if (paramName == "LTE_UL_BANDWIDTH_1") {
								params["LTE_UL_BANDWIDTH_1"] = newValue;
								params["LTE_DL_BANDWIDTH_1"] = newValue;
							} 
						} else {//lgw params
							lgwchange = true;
						}

						//ICIC设置参数
						var icic = /^(LTE_ICIC_.*)|(LTE_CEU_.*)$/;
						if(icic.test(paramName)){
							icicreboot = true;
						}
					}
				}
			});
			//遍历static相关的参数是否修改
			$(".staticItem").each(function() {
				var paramName = $(this).attr("name");
				if (paramName != "UNIT_IP_DNS1" && paramName != "UNIT_IP_DNS2") {
					var newValue = $(this).val();
					var oldValue = $(this).attr("oldValue");
					if(oldValue != newValue) {
						isChangeIpProto = true;
					}
				}
			});
			//如果当前有设置的MME地址，去掉参数params["LTE_SIGLINK_SERVER_LIST"]最后一个字符，
			if (params["LTE_SIGLINK_SERVER_LIST"] != null) {
				var tempMmeAddress = params["LTE_SIGLINK_SERVER_LIST"];
				var mmeAddress = tempMmeAddress.substring(0, tempMmeAddress.length - 1);
				if (mmeAddress == "${S1SIGLINKSERVERLIST}") {
					//删除params对象的LTE_SIGLINK_SERVER_LIST属性
					delete params.LTE_SIGLINK_SERVER_LIST;
				} else {
					params["LTE_SIGLINK_SERVER_LIST"] = mmeAddress;
				}
			}
			//获取用户选择的方式
			var selectProto = $("select[name='UNIT_IP_PROTO']").val();
			//如果现在是static方式
			if(selectProto == "static") {
				//原先是static，现在也是static，且其中的每一个参数都没有变化
				if (params["UNIT_IP_PROTO"] == null && !isChangeIpProto) {
					;
				} else {
					//添加ip，网关，子网掩码参数的值
					var valIsNull = "";
					$(".staticItem").each(function() {
						var attrName = $(this).attr("name");
						var reg = new RegExp("^LGW");
						if(!reg.test(attrName)){
							//默认的两个dns只是显示，不用传递参数
							if (attrName != "UNIT_IP_DNS1" && attrName != "UNIT_IP_DNS2") {
								var attrVal = $(this).val();
								//自定义DNS可以不填
								if (attrVal == "" && attrName != "UNIT_IP_DNS") {
									valIsNull += attrName;
									return false;  //跳出所有循环
								} 
								if (attrName == "UNIT_IP_DNS") {
									var dns1Val = $("input[name='UNIT_IP_DNS1']").val();
									var dns2Val = $("input[name='UNIT_IP_DNS2']").val();
									if (attrVal == "") {
										attrVal = dns1Val + "," + dns2Val;
									} else if (attrVal != dns1Val && attrVal != dns2Val){
										attrVal = $("input[name='UNIT_IP_DNS1']").val() + "," + $("input[name='UNIT_IP_DNS2']").val() + "," + attrVal;
									} else {
										attrVal = dns1Val + "," + dns2Val;
									}
								} 
								params[attrName] = attrVal;
							}
						}
					});
					if (valIsNull.length > 0) {
						if (valIsNull == "UNIT_IP_GATEWAY") {
							$("#UNIT_IP_GATEWAY").blur();
							//return;
						} 
						if (valIsNull == "UNIT_IP_ADDRESS_NEW") {
							$("#UNIT_IP_ADDRESS_NEW").blur();
							//return;
						} 
						if (valIsNull == "UNIT_IP_NET_MASK") {
							$("#UNIT_IP_NET_MASK").blur();
							//return;
						}
						return;
					}
					params["UNIT_IP_PROTO"] = selectProto;
				}
			}

			//判断当前修改的参数是否为空
			var isParamsEmpty = jQuery.isEmptyObject(params);
			var isparamsForCellName = jQuery.isEmptyObject(paramsForeNodeBName);
			//当前参数有修改才下发修改参数的命令
			if(!isParamsEmpty || !isparamsForCellName || lgwchange) {
				params["cell"] = cellCode;
				//如果有设置ip获取方式，则提醒用户此操作会重启
				if (isReboot(params) || lgwchange || icicreboot) {
					if(lgwchange){//process lgw
						paramslgw = finalValidateLgwAndgetParams();
						if("error" == paramslgw){
							return;
						} else {
							params = $.extend({},params,paramslgw);
							//no alert
						}
					}
					//没有重启小站的权限
					if ("${noReboot}" == 1) {
						$.messager.alert(TiShi, "<%=rb.getString("CaoZuoKeNengYinQiZiDongChongQi")%>" + "<%=rb.getString("DouHao")%>" + "<%=rb.getString("MeiYouQuanXian")%>");
						return;
					} else {
						var promptInfo = "<%=rb.getString("CaoZuoKeNengYinQiZiDongChongQi")%>" + "<%=rb.getString("DouHao")%>" + "<%=rb.getString("QueDingChongQiSheBei")%>";
						$.messager.confirm("<%=rb.getString("QueRen")%>", promptInfo, function (r) {
				            if (r) {
				            	$("#winSettingPro").window("open");
				            	//请求修改参数
				    			$.post("${ctx}/cell/param/compactParamConfig.action", params, function(data) {
				    				$("#winSettingPro").window("close");
				    				if (data["success"]) {
				    					$.messager.alert(TiShi, "<%=rb.getString("CaoZuoChengGong")%>");
				    					cellSettingCancel();
				    					//重启基站
				    					var param = {
				    							cell_code: cellCode
				    					};
			    						$.post("${ctx}/cell/cpeinfos/cellReboot.action", param, function (data) {
			    							if (!data["success"]) {
			    								$.messager.alert(TiShi, data["message"]);
			    							} else {
			    								enbvm.refreshList();
			    							}
			    						}, "json");
				    				} else {
				    					$.messager.alert(TiShi, data["message"]);
				    				}
				    			}, "json");
				            }
				        });
					}
				} else {
					$("#winSettingPro").window("open");
					//请求修改参数
					$.post("${ctx}/cell/param/compactParamConfig.action", params, function(data) {
						$("#winSettingPro").window("close");
						if (data["success"]) {
							cellSettingCancel();
							$.messager.alert(TiShi, "<%=rb.getString("CaoZuoChengGong")%>");
							enbvm.refreshList();
						} else {
							$.messager.alert(TiShi, data["message"]);
						}
					}, "json");
				}
			} else {
				$.messager.alert(TiShi, "<%=rb.getString("CanShuZhiMeiYouBianHua")%>");
				return;
			}
		}
	},"json");
}
//添加一个MME地址输入框
function addMMEIpInputText(e) {
	//添加新的输入框
	var oldValue =$("input[id='LTE_SIGLINK_SERVER_LIST']").first().attr("oldValue");
	var $mmeSpan = $("<span style='display:inline-block;visibility:hidden'><%=rb.getString("HeXinWang")%></span>");
	var $mmeDiv = $("<div class='itemDiv'></div>");
	var $mmeInput = $("<input type='text' name='LTE_SIGLINK_SERVER_LIST' class='border border-box item' "
		 + "id='"+Math.random()+"' onblur='validateIPAddress1(event)'  oldValue='"+oldValue+"' style='margin-left:4px'></input>");
	var $mmeButtn = $("<a onclick='removeMMEIpInputText(this)' style='margin-left:3px'>" 
			+ "<img src='${ctx}/skin/${manufacturer}/images/bi/setting_sub.png'/></a>");
	
	$mmeDiv.append($mmeSpan);
	$mmeDiv.append($mmeInput);
	$mmeDiv.append($mmeButtn);
	
	//将新增的输入框插入到plmn参数之前
	var plmnidDiv = $("input[name='LTE_OAM_PLMNID']").parent("div");
	$mmeDiv.insertBefore(plmnidDiv);
	
	//将当前输入框后面的图标改为删除图标，并重新绑定事件
	$(e).children("img").attr("src","${ctx}/skin/${manufacturer}/images/bi/setting_add.png");
	$(e).attr("onclick", "addMMEIpInputText(this)");
}
//删除选中的MME地址输入框
function removeMMEIpInputText(e) {
	$(e).parent("div").remove();
	//如果当前第一个MME输入框的span不显示，将其显示
	var mmeDiv = $("input[name='LTE_SIGLINK_SERVER_LIST']:first");
	var isVisible = mmeDiv.parent("div").children("span").css("visibility");
	if (isVisible == "hidden") {
		mmeDiv.parent("div").children("span").css("visibility","visible");
	}
}

function createIpsecInputDiv(name,filedName,value,disabled){
	if(disabled == true){
		var div = "<div class='itemDiv'><span>"+name+"</span><input type='text' name='"+filedName+"' class='border border-box item' oldValue='"
		+value+"' value='"+value+"' disabled='disabled'/></div>";
	}else{
		var div = "<div class='itemDiv'><span>"+name+"</span><input id='"+filedName+"' onblur='commonChangeTip(event)' type='text' name='"+filedName+"' class='border border-box item'  oldValue='"
	   +value+"' value='"+value+"'/></div>";
	}
	return $(div);
}

function createIpsecInputPasswordDiv(name,filedName,value,disabled){
	var tip = createTitleForIpSec(filedName);
	if(disabled == true){
		var div = "<div class='itemDiv'><span>"+name+"</span><input type='password' name='"+filedName+"' class='border border-box item' oldValue='"
		+value+"' value='"+value+"' disabled='disabled'/></div>";
	}else{
		var div = "<div class='itemDiv'><span>"+name+"</span><input type='password' id='pw' onblur='commonChangeTip(event)' name='"+filedName+"' class='border border-box item'  oldValue='"
	   +value+"' value='"+value+"'/></div>";
	}
	return $(div);
}

function createIpsecSelectDiv(jsonOption,selectedValue,name,filedName){
	// var str = {'name1':1,"name2":2};name1为显示值，1为value
	var tip = createTitleForIpSec(filedName);
	var opts="<option value=''> </option>";
	for(var key in jsonOption){
		if(selectedValue == jsonOption[key]){
			opts = opts+"<option value='"+jsonOption[key]+"' selected='selected'>"+key+"</option>";
		}else{
			opts = opts+"<option value='"+jsonOption[key]+"'>"+key+"</option>";
		}
	}
	var selectHtml="<select name='"+filedName+"' id='AK_cert' class='border border-box item'>"+opts+"</select>";
	if (!selectedValue) {
		selectedValue = "";
	}
	var div = "<div class='itemDiv'><span>"+name+"</span>"+selectHtml+"<input type='hidden' value='"+selectedValue+"' id='OLD_"+filedName+"'>"+
	"<input type='hidden' value='"+selectedValue+"' id='old_AK_cert'></div>";
	return $(div);
}


function validateMaxAndMinVal1(e) {
	var ele = $(e["target"]);
    ele.siblings(".promptTitle").hide();
    validateMaxAndMinVal(e);
	if(!$("#" + ele.attr("id") + "_err").is(':visible')){
	    var currVal = ele.val();
	    var oldValue = ele.attr("oldValue");
	    if(currVal!=oldValue){
	    	$("#" +ele.attr("id")).css({"color":"blue"});
	    	ele.siblings(".promptTitle").show();
	    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValue);
	    }else{
	    	$("#" +ele.attr("id")).css({"color":"black"});
	    	ele.siblings(".promptTitle").hide();
	    }
	    var id = ele.attr("id");
	    if(id=="LTE_FREQ_BAND_INDICATOR"){
	    	$("#LTE_BANDS_SUPPORTED").val(currVal);
	    }
	}
}


function validateMaxAndMinLength1(e){
  	var ele = $(e["target"]);
    var currVal = ele.val();
    var oldValue = ele.attr("oldValue");
    if(currVal!=oldValue){
    	$("#" +ele.attr("id")).css({"color":"blue"});
    }else{
    	$("#" +ele.attr("id")).css({"color":"black"});
    }
    validateMaxAndMinLength(e);
}
function validateHostName(e){
	var ele = $(e["target"]);
    var currVal = ele.val();
    var reg = /^[A-Za-z0-9_\-]{0,48}$/;
    if(currVal!=""){
	    if(!reg.test(currVal)){
	    	 $("#" + ele.attr("id") + "_err").show();
	         ele.addClass("err_border");
	    }else{
	    	 $("#" + ele.attr("id") + "_err").hide();
	         ele.removeClass("err_border");
	    }
    }else{
         ele.removeClass("err_border");
    }
}

function validateIPAddress1(e){
	var ele = $(e["target"]);
    ele.siblings(".promptTitle").hide();
    validateIPAddress(e);
    if(!$("#" + ele.attr("id") + "_err").is(':visible')){
	    var currVal = ele.val();
	    var oldValue = ele.attr("oldValue");
	    if(currVal!=oldValue){
	    	$("#" +ele.attr("id")).css({"color":"blue"});
	    	ele.siblings(".promptTitle").show();
	    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValue);
	    }else{
	    	$("#" +ele.attr("id")).css({"color":"black"});
	    	ele.siblings(".promptTitle").hide();
	    }
    }
}

function validateIPAddress11(e){
	var ele = $(e["target"]);
    ele.siblings(".promptTitle").hide();
    validateIPAddress(e);
    if(!$("#" + ele.attr("id") + "_err").is(':visible')){
	    var currVal = ele.val();
	    var oldValue = ele.attr("oldValue");
	    if(currVal!=oldValue){
	    	$(ele).css({"color":"blue"});
	    }else{
	    	$(ele).css({"color":"black"});
	    }
    }
}

function commonChangeTip(e){
  	var ele = $(e["target"]);
    var currVal = ele.val();
    var oldValue = ele.attr("oldValue");
    if(currVal!=oldValue){
    	$("#" +ele.attr("id")).css({"color":"blue"});
    }else{
    	$("#" +ele.attr("id")).css({"color":"black"});
    }
}

function chageColor(){
	var opts = $("select[id='AK_cert']").find("option");
	var oldValue=$("#old_AK_cert").val();
	var nowValue = $("#AK_cert").val();
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == nowValue){
			$(opts[i]).attr("selected",true);
			if(oldValue!=nowValue){
				$("#AK_cert").css({"color":"blue"});
				$("#AK_cert option").css({"color":"black"});
			}else{
				$("#AK_cert").css({"color":"black"});	
			}
			break;
		}
	}
}
//右键X号收起列表详情
function cellSettingCancel(){
	var settingInput = $("#cellsetting_basic input[type!=hidden]:visible");
	var settingselect = $("#cellsetting_basic select:visible");
	$.each(settingInput,function(index,ele){
		$(ele).val($(ele).attr('oldvalue'))  ;
	})
	$.each(settingselect,function(index,ele){
		$(ele).val($(ele).siblings('input:hidden').val());
	})
    $("#cellsetting_basic").animate({right:'-1200px'},500);
}

function createTitleForIpSec(fieldName){
	<%-- var RSIndex = "<%=rb.getString("RSIndex")%>";
	var RSTunnelName = "<%=rb.getString("RSTunnelName")%>";
	var RSTunnelGateway = "<%=rb.getString("RSTunnelGateway")%>";
	var RSRightSubnet = "<%=rb.getString("RSRightSubnet")%>";
	var RSRightIdentifier = "<%=rb.getString("RSRightIdentifier")%>";
	var RSAuthBy = "<%=rb.getString("RSAuthBy")%>";
	var RSPreSharedKey = "<%=rb.getString("RSPreSharedKey")%>";
	if("ID_1"==fieldName){
		return RSIndex;
	}
	if("TUNNEL_NAME_1"==fieldName){
		return RSTunnelName;
	}
	if("TUNNEL_GATEWAY_1"==fieldName){
		return RSTunnelGateway;
	}
	if("RIGHT_SUBNET_1"==fieldName){
		return RSRightSubnet;
	}
	if("RIGHT_IDENTIFIER_1"==fieldName){
		return RSRightIdentifier;
	}
	if("AUTHBY_1"==fieldName){
		return RSAuthBy;
	}
	if("PRE_SHARED_KEY_1"==fieldName){
		return RSPreSharedKey;
	}  --%>
}
function lgwEnableChange(ele){
	var enaleFlag =	$(ele).val();
	var lgwmode = $("#intelTdd_LGW_MODE").val();
	var staticAddressEnaleFlag = $("#intelTdd_LGW_STATIC_ADDRESS").val();
	if("0" == enaleFlag){
		$("div[name='LGW']").hide();
	} else {
		$("div[name='LGW']").show();
		if("NAT" == lgwmode){
			$("#intelTdd_LGW_IP_POOL").parent().show();
			$("#intelTdd_LGW_IP_POOL_NETMASK").parent().show();
			$("#intelTdd_LGW_STATIC_ADDRESS").parent().hide();
			$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().hide();
			$("input[name='LGW_IMSI']").parent().hide();
			$("#lgwiprange").hide();
		} else if("Router" == lgwmode){
			$("#intelTdd_LGW_IP_POOL").parent().show();
			$("#intelTdd_LGW_IP_POOL_NETMASK").parent().show();
			$("#intelTdd_LGW_STATIC_ADDRESS").parent().show();
			if("0" == staticAddressEnaleFlag){
				$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().hide();
				$("input[name='LGW_IMSI']").parent().hide();
				$("#lgwiprange").hide();
			} else {
				$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().show();
				$("input[name='LGW_IMSI']").parent().show();
				$("#lgwiprange").show();
				$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").blur();
				$("#intelTdd_LGW_STATIC_IP_CONFIG_END").blur();
			}
		} else if("Bridge" == lgwmode){
			$("#intelTdd_LGW_IP_POOL").parent().hide();
			$("#intelTdd_LGW_IP_POOL_NETMASK").parent().hide();
			$("#intelTdd_LGW_STATIC_ADDRESS").parent().hide();
			$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().hide();
			$("input[name='LGW_IMSI']").parent().hide();
			$("#lgwiprange").hide();
		} 
	}
}
function lgwModeChange(ele){
	var lgwmode = $(ele).val();
	var staticipEnable = $("#intelTdd_LGW_STATIC_ADDRESS").val();
	if("NAT" == lgwmode){
		$("#intelTdd_LGW_IP_POOL").parent().show();
		$("#intelTdd_LGW_IP_POOL_NETMASK").parent().show();
		$("#intelTdd_LGW_STATIC_ADDRESS").parent().hide();
		$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().hide();
		$("input[name='LGW_IMSI']").parent().hide();
		$("#lgwiprange").hide();
		$(ele).siblings(".promptTitle").hide();
	} else if("Router" == lgwmode){
		$("#intelTdd_LGW_STATIC_ADDRESS").parent().show();
		$("#intelTdd_LGW_IP_POOL").parent().show();
		$("#intelTdd_LGW_IP_POOL_NETMASK").parent().show();
		$(ele).siblings(".promptTitle").show();
    	$(ele).siblings(".promptTitle").find(".unmodiValue").text("NAT");
		if("0" == staticipEnable){
			$("input[name='LGW_IMSI']").parent().hide();
			$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().hide();
			$("#lgwiprange").hide();
		} else {
			$("input[name='LGW_IMSI']").parent().show();
			$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().show();
			$("#lgwiprange").show();
		}
	} else if("Bridge" == lgwmode){
		$("#intelTdd_LGW_IP_POOL").parent().hide();
		$("#intelTdd_LGW_IP_POOL_NETMASK").parent().hide();
		$("#intelTdd_LGW_STATIC_ADDRESS").parent().hide();
		$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().hide();
		$("input[name='LGW_IMSI']").parent().hide();
		$("#lgwiprange").hide();
		$(ele).siblings(".promptTitle").show();
    	$(ele).siblings(".promptTitle").find(".unmodiValue").text("NAT");
	} 
}
function staticIPEnable(ele){
	var enaleFlag =	$(ele).val();
	if("0" == enaleFlag){
		$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().hide();
		$("input[name='LGW_IMSI']").parent().hide();
		$("#lgwiprange").hide();
	} else {
		$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().show();
		$("input[name='LGW_IMSI']").parent().show();
		$("#lgwiprange").show();
	}
}

function ippoolonblur(e){
	var ele = $(e["target"]);
	var retflag = validateLgwIp(e);
	if(!retflag){
		ele.siblings(".promptTitle").hide();
		return;
	}
	var ipvalue = $(ele).val();
	var ippoolmask = $("#intelTdd_LGW_IP_POOL_NETMASK").val();
	
	if(!$("#" + ele.attr("id") + "_err").is(':visible')){
		var lowip = getLowAddr(ipvalue,ippoolmask);
		var highip = getHighAddr(ipvalue,ippoolmask);
		var hostnumber = getHostNumber(ippoolmask);
		//TIP MESSAGE
		var lgwmode = $("#intelTdd_LGW_MODE").val();
		var lgwipenable = $("#intelTdd_LGW_STATIC_ADDRESS").val();
		if("Router" == lgwmode && "1" == lgwipenable ){
			$("#lgwiprange span").html(lowip + "-" + highip);
			$("#intelTdd_LGW_STATIC_IP_CONFIG_err span").html(lowip + "-" + highip);
			//提示框赋值
		}
		var oldValue = ele.attr("oldValue");
		if(ipvalue!=oldValue){
		   ele.siblings(".promptTitle").show();
		   ele.siblings(".promptTitle").find(".unmodiValue").text(oldValue);
		}else{
		    	ele.siblings(".promptTitle").hide();
		}
	}
	if($("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").val() != ''){
		$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").blur();
	}
	if($("#intelTdd_LGW_STATIC_IP_CONFIG_END").val() != ''){
		$("#intelTdd_LGW_STATIC_IP_CONFIG_END").blur();
	}
	var IMCIIPinput = $("input[name=LGW_IMSI_IP]");
	$(IMCIIPinput).each(function(index,ele){
		if($(ele).val() != ''){
			$(ele).blur();
		}
	});
}


function validateipandrange(e){
	var ele = $(e["target"]);
	var retflag = validStaticAdressRange(e);
	if(!retflag){
		ele.siblings(".promptTitle").hide();
		return;
	}else{
		if(!$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").hasClass('err_border') && !$("#intelTdd_LGW_STATIC_IP_CONFIG_END").hasClass('err_border')){
			ele.siblings(".promptTitle").show();
		}
	}
	var ipvalue = $(ele).val();
	var ip = $("#intelTdd_LGW_IP_POOL").val();
	var netmask = $("#intelTdd_LGW_IP_POOL_NETMASK").val();
	var ipstart = getLowAddr(ip,netmask);
	var ipend = getHighAddr(ip,netmask);
	var compareFalg = compareIp(ipvalue,ipstart,ipend);
	if(!compareFalg){
		ele.siblings(".promptTitle").hide();
		$("#intelTdd_LGW_STATIC_IP_CONFIG_err").show();
        ele.addClass("err_border");
	} else {
		if(!$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").hasClass('err_border') && !$("#intelTdd_LGW_STATIC_IP_CONFIG_END").hasClass('err_border')){
			ele.siblings(".promptTitle").show();
			$("#intelTdd_LGW_STATIC_IP_CONFIG_err").hide();
		}
        ele.removeClass("err_border");
	}
	var IMCIIPinput = $("input[name=LGW_IMSI_IP]");
	$(IMCIIPinput).each(function(index,ele){
		if($(ele).val() != ''){
			$(ele).blur();
		}
	});
}

function validateipandrangeforbinding(e){
	var ele = $(e["target"]);
	var ipvalue = $(ele).val();
	
	var retflag = validateLgwIp(e);
	if(!retflag){
		if(ipvalue != ''){
			if($("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").val() == ''){
				$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").blur();
			}
			if($("#intelTdd_LGW_STATIC_IP_CONFIG_END").val() == ''){
				$("#intelTdd_LGW_STATIC_IP_CONFIG_END").blur();
			}
		}
		return;
	}
	var ipstart = $("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").val();
	var ipend = $("#intelTdd_LGW_STATIC_IP_CONFIG_END").val();
	var ip = $("#intelTdd_LGW_IP_POOL").val();
	var netmask = $("#intelTdd_LGW_IP_POOL_NETMASK").val();
	var ipstart1 = getLowAddr(ip,netmask);
	var ipend1 = getHighAddr(ip,netmask);
	var compareFalg1 = compareIp(ipstart,ipstart1,ipend1);
	var compareFalg2 = compareIp(ipend,ipstart1,ipend1);
	
	var flag = (compareFalg1 && compareFalg2);
	
	var compareFalg = compareIp(ipvalue,ipstart,ipend);
	if(!compareFalg || !flag){
		$("#" + ele.attr("id") + "_err").show();
        ele.addClass("err_border");
	} else {
		$("#" + ele.attr("id") + "_err").hide();
        ele.removeClass("err_border");
	}
}

function validStaticAdressRange(e){
	var ele = $(e["target"]);
    var currVal = ele.val();
    var oldValue = ele.attr("oldValue");
    if(currVal!=oldValue){
    	$(ele).css({"color":"blue"});
    }else{
    	$(ele).css({"color":"black"});
    }
    var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
	var ele = $(e["target"]);
	if (ele.val().length == 0) {
		var must = ele.attr("must");
		if (must == "1") {
			$("#intelTdd_LGW_STATIC_IP_CONFIG_err").show();
	        ele.addClass("err_border");
		} else {
			$("#intelTdd_LGW_STATIC_IP_CONFIG_err").show();
	        ele.removeClass("err_border");
		}
		return false;
	}
	
	if (reg.test(ele.val())) {// 格式正确
		$("#intelTdd_LGW_STATIC_IP_CONFIG_err").show();
        ele.removeClass("err_border");
	} else {
		$("#intelTdd_LGW_STATIC_IP_CONFIG_err").show();
        ele.addClass("err_border");
        return false;
	}
	return true;
}
function imsionblur(e){
	var ele = $(e["target"]);
	var imsivalue = $(ele).val();
	if(!imsivalue){
		imsivalue = "";
	}
    var oldValue = ele.attr("oldValue");
    if(imsivalue!=oldValue){
    	$(ele).css("color","blue");
    }else{
    	$(ele).css("color","black");
    }
	if(imsivalue.length != 15){
		$("#" + ele.attr("id") + "_err").show();
        ele.addClass("err_border");
	} else {
		$("#" + ele.attr("id") + "_err").hide();
        ele.removeClass("err_border");
	}
}
function validateLgwIp(e){
	var ele = $(e["target"]);
    var currVal = ele.val();
    var oldValue = ele.attr("oldValue");
    if(currVal!=oldValue){
    	$(ele).css({"color":"blue"});
    }else{
    	$(ele).css({"color":"black"});
    }
    var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
	var ele = $(e["target"]);
	
	if (ele.val().length == 0) {
		var must = ele.attr("must");
		if (must == "1") {
			$("#" + ele.attr("id") + "_err").show();
	        ele.addClass("err_border");
		} else {
			$("#" + ele.attr("id") + "_err").hide();
	        ele.removeClass("err_border");
		}
		return false;
	}
	
	if (reg.test(ele.val())) {// 格式正确
		$("#" + ele.attr("id") + "_err").hide();
        ele.removeClass("err_border");
	} else {
		$("#" + ele.attr("id") + "_err").show();
        ele.addClass("err_border");
        return false;
	}
	return true;
}      
function getLowAddr(ip, netMask) {
    var lowAddr = "";
    var ipArray = new Array();
    var netMaskArray = new Array();

    if (4 != ip.split(".").length || netMask == "") {
        return "";
    }
    for (var i = 0; i < 4; i++) {
        ipArray[i] = ip.split(".")[i];
        netMaskArray[i] = netMask.split(".")[i];
        if ((ipArray[i] > 255) || (ipArray[i] < 0) || (netMaskArray[i] > 255)
                && (netMaskArray[i] < 0)) {
            return "";
        }
        ipArray[i] = ipArray[i] & netMaskArray[i];
    }

    for (var i = 0; i < 4; i++) {
        if (i == 3) {
            ipArray[i] = ipArray[i] + 1;
        }
        if (lowAddr == "") {
            lowAddr += ipArray[i];
        } else {
            lowAddr += "." + ipArray[i];
        }
    }
    return lowAddr;
}

function getHighAddr(ip,netMask){
    var lowAddr = getLowAddr(ip,netMask);
    var hostNumber = getHostNumber(netMask);
    if(lowAddr == "" || hostNumber == 0)
  {
        return "";
    }

    var lowAddrArray = new Array();
    for(var i = 0; i < 4; i++)
  {
        lowAddrArray[i] = lowAddr.split(".")[i];
        if(i == 3)
    {
            lowAddrArray[i] = Number(lowAddrArray[i] - 1);
        }
    }
    lowAddrArray[3] = lowAddrArray[3] + Number(hostNumber - 1);
    //alert(lowAddrArray[3]);
    if(lowAddrArray[3] > 255)
  {
        var k = parseInt(lowAddrArray[3] / 256);
        //alert(k);
        lowAddrArray[3] = lowAddrArray[3] % 256;
        //alert(lowAddrArray[3]);
        lowAddrArray[2] = Number(lowAddrArray[2]) + Number(k);
        //alert(lowAddrArray[2]);
        if(lowAddrArray[2] > 255)
    {
            k = parseInt(lowAddrArray[2] / 256);
            lowAddrArray[2] = lowAddrArray[2] % 256;
            lowAddrArray[1] = Number(lowAddrArray[1]) + Number(k);
            if(lowAddrArray[1] > 255)
      {
                k = parseInt(lowAddrArray[1] / 256);
                lowAddrArray[1] = lowAddrArray[1] % 256;
                lowAddrArray[0] = Number(lowAddrArray[0]) + Number(k);
            }
        }
    }

    var highAddr = "";
 for(var i = 0; i < 4; i++)
  {
        if(i == 3)
    {
      lowAddrArray[i] = lowAddrArray[i] - 1;
        }
        if(highAddr == "")
    {
            highAddr = lowAddrArray[i];

        }else{
            highAddr += "." + lowAddrArray[i];
        }
    }

    return highAddr;
}
function getHostNumber(netMask){
    var hostNumber = 0;
    var netMaskArray = new Array();
    for(var i = 0; i < 4; i++)
  {
        netMaskArray[i] = netMask.split(".")[i];
        if(netMaskArray[i] < 255)
    {
            hostNumber = Math.pow(256,3-i) * (256 - netMaskArray[i]);
            break;
        }
    }

    return hostNumber;
}
var i =2;
var imsiIP_fdd = 0;
//添加一个MME地址输入框
function intelTdd_addISMIAndIPInputText(e) {
	if(i == 10){
		return;
	}
	imsiIP_fdd += 1;
	//添加新的输入框
	var oldValue = "";//$("input[id='LTE_SIGLINK_SERVER_LIST']").first().attr("oldValue");
	var $mmeSpan = $("<span style='display:block;visibility:hidden'><%=rb.getString("IMSIBangDing")%></span>");
	var $mmeDiv = $("<div class='itemDiv staticDiv lgw' name='LGW' id='"+i+"'></div>");
	var $mmeInput = $("<input type='text' name='LGW_IMSI' class='border border-box itemStatisticInput' style='width:169px;height:26px;margin-left:0px;padding-left:10px' "  + " title='<%=rb.getString("LGWImsiTiShi")%>'"
		 + "id='intelTdd_LGW_IMSI_"+imsiIP_fdd+"' onblur='imsionblur(event)'  oldValue='"+oldValue+"' ></input>"
		 +" - "
		 + "<input type='text' name='LGW_IMSI_IP' class='border border-box itemStatisticInput' style='width:169px;height:26px;padding-left:10px' "  + " title='<%=rb.getString("LGWIpTiShi")%>'"
		 + "id='intelTdd_LGW_IMSI_IP_"+imsiIP_fdd+"' onblur='validateipandrangeforbinding(event)'  oldValue='"+oldValue+"' ></input>"
	);
	var $mmeButtn = $("<a onclick='intelTdd_removeIMSIAndIPInputText(this)' style='margin-left:3px' >" 
			+ "<img src='${ctx}/skin/${manufacturer}/images/bi/setting_sub.png'/></a>"); 
	var $tip = $("<div class='errorTitle' id='intelTdd_LGW_IMSI_"+ imsiIP_fdd +"_err'><%=rb.getString("LGWImsiTiShi")%></span></div>"+
		    "<div class='errorTitle' id='intelTdd_LGW_IMSI_IP_"+ imsiIP_fdd +"_err'><%=rb.getString("LGWIpTiShi")%></span></div>");
	
	$mmeDiv.append($mmeSpan);
	$mmeDiv.append($mmeInput);
	$mmeDiv.append($mmeButtn);
	$mmeDiv.append($tip);
	$mmeDiv.addClass("clearBoth");
	//将新增的输入框插入到plmn参数之前
	var plmnidDiv = $("#after_log_ismi_ip");
	$mmeDiv.insertBefore(plmnidDiv); 
	
	//将当前输入框后面的图标改为删除图标，并重新绑定事件
	//$(e).children("img").attr("src","${ctx}/skin/${manufacturer}/images/bi/setting_add.png");
	//$(e).attr("onclick", "intelTdd_addISMIAndIPInputText(this)");
	i++;
}
//删除选中的MME地址输入框
function intelTdd_removeIMSIAndIPInputText(e) {
	$(e).parent("div").remove();
	//如果当前第一个MME输入框的span不显示，将其显示
	var mmeDiv = $("#intelTdd_LGW_IMSI");
	var isVisible = mmeDiv.parent("div").children("span").css("visibility");
	if (isVisible == "hidden") {
		mmeDiv.parent("div").children("span").css("visibility","visible");
	}
	i--;
}
/**
 * 仅适用于lgw配置
 */
 function compareIp(ipvalue,startip,endip){
	var ipArray = ipvalue.split(".");
	var startipArray = startip.split(".");
	var endipArray = endip.split(".");
	for(var i = 0; i < 4; i++){
		if(i<3){
			if(ipArray[i] != startipArray[i]){
				return false;
			}
		} else {
			if(!((ipArray[3] * 1) >= (startipArray[3] * 1) && (ipArray[3] * 1) <= (endipArray[3]) * 1)){
				return false;
			}
		}
	}
	return true;
}
$("#intelTdd_LGW_STATIC_ADDRESS").bind("change",function(){
		//移除之前被选中的元素
		//$("#intelTdd_LGW_STATIC_ADDRESS option:selected").removeAttr("selected");
		//选中新选择的元素
		var opts = $("#intelTdd_LGW_STATIC_ADDRESS").find("option");
		var oldValue=$("#intelTdd_OLD_LGW_STATIC_ADDRESS").attr("oldValue");
		for(var i=0;i<opts.length;i++){
			if(opts[i].value == this.value){
				$(opts[i]).attr("selected",true);
				if(oldValue!=this.value){
					$("#intelTdd_LGW_STATIC_ADDRESS").css({"color":"blue"});
					$("#intelTdd_LGW_STATIC_ADDRESS option").css({"color":"black"});
				}else{
					$("#intelTdd_LGW_STATIC_ADDRESS").css({"color":"black"});	
				}
				break;
			}
		}
		$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").blur();
		$("#intelTdd_LGW_STATIC_IP_CONFIG_END").blur();
		var IMCIIPinput = $("input[name=LGW_IMSI_IP]");
		$(IMCIIPinput).each(function(index,ele){
			$(ele).blur();
		});
		$("#intelTdd_LGW_IP_POOL").blur();
});
 function initlgw(){
		var LGW_SWITCH = "${LGW_SWITCH}";
		var LGW_TRANSFER_MODE	 = "${LGW_TRANSFER_MODE}";
		var LGW_IFNAME	 = "${LGW_IFNAME}";
		var LGW_START_UE_ADDRD = "${LGW_START_UE_ADDRD}";
		var LGW_NET_MASK	 = "${LGW_NET_MASK}";
		var LGW_STATIC_IP_ADDR_SWITCH	 = "${LGW_STATIC_IP_ADDR_SWITCH}";
		var LGW_FIRST_STATIC_IP_ADDRESS = "${LGW_FIRST_STATIC_IP_ADDRESS}";
		var LGW_LAST_STATIC_IP_ADDRESS = "${LGW_LAST_STATIC_IP_ADDRESS}";
		var LGW_IMSI_IP_LIST = "${LGW_IMSI_IP_LIST}";
		
		if(LGW_START_UE_ADDRD && LGW_START_UE_ADDRD!="" && LGW_NET_MASK && LGW_NET_MASK!=""){
			var lowip = getLowAddr(LGW_START_UE_ADDRD,LGW_NET_MASK);
			var highip = getHighAddr(LGW_START_UE_ADDRD,LGW_NET_MASK);
			$("#lgwiprange span").html(lowip + "-" + highip);
		}
		
		$("#intelTdd_LGW_ENABLE").val(LGW_SWITCH);
		$("#intelTdd_LGW_MODE").val(LGW_TRANSFER_MODE);
		$("#intelTdd_LGW_INTERFACE_BINDING").val(LGW_IFNAME);
		$("#intelTdd_LGW_IP_POOL").val(LGW_START_UE_ADDRD);
		$("#intelTdd_LGW_IP_POOL_NETMASK").val(LGW_NET_MASK);
		$("#intelTdd_LGW_STATIC_ADDRESS").val(LGW_STATIC_IP_ADDR_SWITCH);
		$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").val(LGW_FIRST_STATIC_IP_ADDRESS);
		$("#intelTdd_LGW_STATIC_IP_CONFIG_END").val(LGW_LAST_STATIC_IP_ADDRESS);
		if(LGW_IMSI_IP_LIST && LGW_IMSI_IP_LIST.length > 1){
			var imsiandipArray = LGW_IMSI_IP_LIST.split(",");
			for(var j =1;j<imsiandipArray.length;j++){
				intelTdd_addISMIAndIPInputText($("a"));
			}
			 
			$("#lgw .lgw").each(function(i){
				if(i<imsiandipArray.length){
					var xx = imsiandipArray[i];
					var imsi = imsiandipArray[i].split("+")[0];
					var ip = imsiandipArray[i].split("+")[1];
					var id = $(this).attr("id");
					$("#" + id + " input[name='LGW_IMSI']").val(imsi);
					$("#" + id + " input[name='LGW_IMSI']").attr("oldValue",imsi)
					$("#" + id + " input[name='LGW_IMSI_IP']").val(ip);
					$("#" + id + " input[name='LGW_IMSI_IP']").attr("oldValue",ip);
				}
			});
			
		}
		
		if("" == LGW_STATIC_IP_ADDR_SWITCH){
			$("#intelTdd_LGW_STATIC_ADDRESS").val("0");
			$("#intelTdd_OLD_LGW_STATIC_ADDRESS").val("0");
		}
		
		if("" == LGW_SWITCH || !LGW_SWITCH){
			LGW_SWITCH = "0";
			$("#intelTdd_LGW_ENABLE").val(LGW_SWITCH);
			$("#intelTdd_OLD_LGW_ENABLE").val(LGW_SWITCH);
		}
		if("" == LGW_TRANSFER_MODE || !LGW_TRANSFER_MODE){
			LGW_TRANSFER_MODE = "NAT";
			$("#intelTdd_LGW_MODE").val(LGW_TRANSFER_MODE);
			$("#intelTdd_OLD_LGW_MODE").val(LGW_TRANSFER_MODE);
		}
		if("" == LGW_IFNAME || !LGW_IFNAME){
			LGW_IFNAME = "wan";
			$("#intelTdd_LGW_INTERFACE_BINDING").val(LGW_IFNAME);
			$("#intelTdd_OLD_LGW_INTERFACE_BINDING").val(LGW_IFNAME);
		}
		if("" == LGW_START_UE_ADDRD || !LGW_START_UE_ADDRD){
			LGW_START_UE_ADDRD = "10.0.0.1";
			$("#intelTdd_LGW_IP_POOL").val(LGW_START_UE_ADDRD);
			$("#intelTdd_LGW_IP_POOL").attr("oldValue",LGW_START_UE_ADDRD);
		}
		if("" == LGW_NET_MASK || !LGW_NET_MASK){
			LGW_NET_MASK = "255.255.255.0";
			$("#intelTdd_LGW_IP_POOL_NETMASK").val(LGW_NET_MASK);
			$("#intelTdd_OLD_LGW_IP_POOL_NETMASK").val(LGW_NET_MASK);
			$("#intelTdd_OLD_LGW_IP_POOL_NETMASK").attr("oldValue",LGW_NET_MASK);
		}
		if("0" == LGW_SWITCH){
			$("div[name='LGW']").hide();
		} else {
			$("div[name='LGW']").show();
			if("NAT" == LGW_TRANSFER_MODE){
				$("#intelTdd_LGW_IP_POOL").parent().show();
				$("#intelTdd_LGW_IP_POOL_NETMASK").parent().show();
				$("#intelTdd_LGW_STATIC_ADDRESS").parent().hide();
				$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().hide();
				$("input[name='LGW_IMSI']").parent().hide();
				$("#lgwiprange").hide();
				
			} else if("Router" == LGW_TRANSFER_MODE){
				$("#intelTdd_LGW_IP_POOL").parent().show();
				$("#intelTdd_LGW_IP_POOL_NETMASK").parent().show();
				$("#intelTdd_LGW_STATIC_ADDRESS").parent().show();
				
				if("0" == LGW_STATIC_IP_ADDR_SWITCH){
					$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().hide();
					$("input[name='LGW_IMSI']").parent().hide();
					$("#lgwiprange").hide();
				} else {
					$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().show();
					$("input[name='LGW_IMSI']").parent().show();
					$("#lgwiprange").show;
				}
			} else if("Bridge" == LGW_TRANSFER_MODE){
				$("#intelTdd_LGW_IP_POOL").parent().hide();
				$("#intelTdd_LGW_IP_POOL_NETMASK").parent().hide();
				$("#intelTdd_LGW_STATIC_ADDRESS").parent().hide();
				$("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").parent().hide();
				$("input[name='LGW_IMSI']").parent().hide();
				$("#lgwiprange").hide();
			} 
		}
	}
 function finalValidateLgwAndgetParams(){
		var params = {};
		var lgwImsiAndIP = "";
		var lgwEnable = $("#intelTdd_LGW_ENABLE").val();
		var lgwMode = $("#intelTdd_LGW_MODE").val();
		var lgwInterfaceBinding = $("#intelTdd_LGW_INTERFACE_BINDING").val();
		var lgwIpPool = $("#intelTdd_LGW_IP_POOL").val();
		var lgwIpPoolNetMask = $("#intelTdd_LGW_IP_POOL_NETMASK").val();
		var lgwStaticAddressEnable = $("#intelTdd_LGW_STATIC_ADDRESS").val();
		var lgwStaticAddressRangeStart = $("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").val();
		var lgwStaticAddressRangeEnd = $("#intelTdd_LGW_STATIC_IP_CONFIG_END").val();
		
		//validate
		if(lgwEnable == "1"){
			if("NAT" == lgwMode){
				var retFlag = validateLgwIpPool(lgwIpPool);
				if(!retFlag){
					return "error";
				}
			} else if("Router" == lgwMode){
				var retFlag = validateLgwIpPool(lgwIpPool);
				if(!retFlag){
					return "error";
				}
				if(lgwStaticAddressEnable == "1"){
					var retFlag = validateLgwStaticIpRage(lgwIpPool,lgwIpPoolNetMask,lgwStaticAddressRangeStart,lgwStaticAddressRangeEnd);
					if(!retFlag){
						return "error";
					}
					var retFlag = validateLgwImsiAndIp(lgwStaticAddressRangeStart,lgwStaticAddressRangeEnd);
					if(!retFlag){
						return "error";
					}
				}
			}
		}
		
		
			//pass validate
			params.lgwEnable = lgwEnable;
			params.lgwMode = lgwMode;
			params.lgwInterfaceBinding = "eth2";
			if(lgwEnable == "1"){
				if("NAT" == lgwMode){
					params.lgwMode = "0";
					params.lgwIpPool =lgwIpPool;
					params.lgwIpPoolNetMask =lgwIpPoolNetMask;
				} else if("Router" == lgwMode){
					params.lgwMode = "1";
					params.lgwIpPool =lgwIpPool;
					params.lgwIpPoolNetMask =lgwIpPoolNetMask;
					params.lgwStaticAddressEnable = lgwStaticAddressEnable;
					if(lgwStaticAddressEnable == "1"){
						params.lgwStaticAddressRangeStart = lgwStaticAddressRangeStart;
						params.lgwStaticAddressRangeEnd = lgwStaticAddressRangeEnd;
						//获取IMSI +　IP
						$("#lgw .lgw").each(function(i){
							var id = $(this).attr("id");
							var imsi = $("#" + id + " input[name='LGW_IMSI']").val();
							var ip = $("#" + id + " input[name='LGW_IMSI_IP']").val();
							lgwImsiAndIP = lgwImsiAndIP  +  imsi + "+" + ip + ",";
						});
						lgwImsiAndIP = lgwImsiAndIP.substring(0,lgwImsiAndIP.length-1);
						params.lgwImsiAndIP = lgwImsiAndIP;
						var retFflag = validateImisAndIpRepeat(lgwImsiAndIP);
						if(!retFflag){
							return "error";
						}
					}
				} else {
					params.lgwMode = "2";
				}
			} else {
				params = {};
				params.lgwEnable = "0";
			}
		return params;
	}
 function validateLgwIpPool(ippool){
		var regip = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
		if(!regip.test(ippool)){
			$.messager.alert(TiShi, "<%=rb.getString("LGWIpPoolGeShiCuoWu")%>");
			return false;
		}
		return true;
	} 


	function validateLgwStaticIpRage(lgwIpPool,lgwIpPoolNetMask,lgwStaticAddressRangeStart,lgwStaticAddressRangeEnd){
		var ipstart = getLowAddr(lgwIpPool,lgwIpPoolNetMask);
		var ipend = getHighAddr(lgwIpPool,lgwIpPoolNetMask);
		if(!lgwStaticAddressRangeStart || !lgwStaticAddressRangeEnd){
			$.messager.alert(TiShi, "<%=rb.getString("QingTianXieLGWJingTaiIPFanWei")%>");
			return false;
		}
		var compareFalg1 = compareIp(lgwStaticAddressRangeStart,ipstart,ipend);
		var compareFalg2 = compareIp(lgwStaticAddressRangeEnd,ipstart,ipend);
		if(!compareFalg1){
			$.messager.alert(TiShi, "<%=rb.getString("LGWJingTaiIPDiZhiYueJie")%>");
			return false;
		} else if(!compareFalg2){
			$.messager.alert(TiShi, "<%=rb.getString("LGWJingTaiIPDiZhiYueJie")%>");
			return false;
		}
		
		if((lgwStaticAddressRangeStart.split(".")[3])*1 > (lgwStaticAddressRangeEnd.split(".")[3])*1){
			$.messager.alert(TiShi, "<%=rb.getString("LGWJingTaiIPDiZhiQiShiIPYingXiaoYuJieShuIP")%>");
			return false;
		}
		return true;
	}


	function validateLgwImsiAndIp(ipstart,ipend){
		var flag = true;
		$("#lgw .lgw").each(function(i){
			var id = $(this).attr("id");
			var imsi = $("#" + id + " input[name='LGW_IMSI']").val();
			var ip = $("#" + id + " input[name='LGW_IMSI_IP']").val();
			if(!imsi){
				imsi = "";
			}
			if(!ip){
				ip = "";
			}
			
			if(imsi.length == 0 && ip.length == 0){
				return true;
			} else if(imsi.length != 0 && ip.length == 0){
				$.messager.alert(TiShi,"<%=rb.getString("QingBangDingIP")%>");
				flag = false;
				return false;
			} else if(imsi.length == 0 && ip.length != 0){
				$.messager.alert(TiShi,"<%=rb.getString("QingBangDingImsi")%>");
				flag = false;
				return false;
			}
				
			if(imsi){
				imsi = imsi.replace(/\s/g,"").trim();
				if(imsi.length != 15){
					$.messager.alert(TiShi, imsi + "," + "<%=rb.getString("LGWImsiChangDuCuoWu")%>");
					flag = false;
					return false;
				}
			} else {
				$.messager.alert(TiShi, "<%=rb.getString("LGWImsiChangDuCuoWu")%>");
				flag = false;
				return false;
			}
			if(ip){
				var regip = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				if(!regip.test(ip)){
					$.messager.alert(TiShi, ip + "," + "<%=rb.getString("LGWIpBangDingIPGeShiCuoWu")%>");
					flag = false;
					return false;
				}
				var compareFalg1 = compareIp(ip,ipstart,ipend);
				if(!compareFalg1){
					$.messager.alert(TiShi, ip +  ","  + "<%=rb.getString("LGWIpBangDingIPYueJie")%>");
					flag = false;
					return false;
				}
				
			} else {
				$.messager.alert(TiShi, "<%=rb.getString("LGWIpBangDingIPGeShiCuoWu")%>");
				flag = false;
				return false;
			} 
		});
		return flag;
	}

	function validateImisAndIpRepeat(lgwImsiAndIP){
		var flag = true;
		if(!lgwImsiAndIP || "" == lgwImsiAndIP){
			return true;
		} else {
			var groupImsiAndIpArray = lgwImsiAndIP.replace(/\"/g,"").trim().split(",");
			for(var i = 0; i < groupImsiAndIpArray.length; i++){
				var imsi = groupImsiAndIpArray[i].split("+")[0];
				var ip = groupImsiAndIpArray[i].split("+")[1];
				for(var j = 0; j < groupImsiAndIpArray.length; j++){
					if(j != i){
						var imsi1 = groupImsiAndIpArray[j].split("+")[0];
						var ip1 = groupImsiAndIpArray[j].split("+")[1];
						if(imsi == imsi1){
							$.messager.alert(TiShi, imsi + "," + "<%=rb.getString("LGWIMSIBangDingChongFu")%>");
							flag = false;
							return false;
						}
						if(ip == ip1){
							$.messager.alert(TiShi, ip + ","  + "<%=rb.getString("LGWIPBangDingChongFu")%>");
							flag = false;
							return false;
						}
					}
				}
			}
		}
		return flag;	
	}
	//重启参数判断
	function isReboot(ele){
		if(ele["LTE_CELL_ECI"] != null || ele["LTE_DL_EARFCN"] != null  ||  ele["LTE_FREQ_BAND_INDICATOR"] != null  || ele["LTE_OAM_PLMNID"] != null  || ele["LTE_PB"] != null  || ele["LTE_PHY_CELLID_LIST"] != null  || ele["LTE_SIGLINK_SERVER_LIST"] != null  || 
				ele["LTE_TAC"] != null || ele["LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS"] != null  || ele["LTE_TDD_SUBFRAME_ASSIGNMENT"] != null  || ele["LTE_UL_BANDWIDTH"] != null  || ele["LTE_UL_EARFCN"] != null){
			return true;
		}else{
			return false;
		}
	}	
</script>