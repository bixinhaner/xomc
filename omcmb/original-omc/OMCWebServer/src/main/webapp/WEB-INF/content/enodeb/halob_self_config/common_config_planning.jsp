<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>
<style>		
.wirelessSetting{
	position : absolute;
	top : 50px;
	bottom : 10px;
	left : 20px;
	overflow : auto;
}
.itemDiv{
		width:375px;
		height:92px;
		float:left;
		/* margin-right:30px; */
		margin-left:40px;
} 
.wirelessSetting select:disabled{
	background:#EAF1F4;
}
.clearBoth{
	float : none !important;
	clear : both;
	margin-right : 400px !important;
}
.selfConfigSucTip{
	display:inline-block;
	min-width:200px;
	height:38px;
	line-height:38px;
	padding : 0 15px 0 50px;
	margin-left:35px;
	color:#508D9B;
	font-size:16px;
	font-weight:bold;
	background:#E9FBFF url("${ctx}/images/success.png") no-repeat 16px center;
}
.errorTitle{
	height:26px;
	line-height:26px;
	min-width:50px;
	font-size:12px;
	color:red;
	display : none;
}
.itemDiv > span:nth-child(1){
	width:230px;
}
.ipsecChannel{
	position: absolute;
    background: #FFF;
    top: 0px;
    clear: both;
    display: none;
}
.saveTip{
	height:26px;
	line-height:26px;
	min-width:50px;
	font-size:12px;
	color:#4c6778;
	display:none;
}
</style>
<div class="slidebarTitleDiv">
	<ul class="slidebarTitleContainer" style="margin-left:0;padding-left:0;">
		<li class="default" id="commonConfigTitle"></li>
	</ul>
	<div class="tableDiv titleIcon_close" onclick="closeCommonConfig();" style="position:absolute;right:25px;top:15px;"></div>
</div>
<div class="wirelessSetting" style="padding: 0 0 0 20px;display:block">
	<div class="omcPageTitleDiv" style="margin-bottom: 20px;">
		<ul class="omcPageTitleContainer">
			<li class="default"><%=rb.getString("JiChuPeiZhi")%></li>
		</ul>
	</div>
	<div id="basicCommonConfig">
		<div class="itemDiv" style="">
			<span><%=rb.getString("ZhiChiPinDuan")%></span>
			<input type="text" name="basic_BANDS_SUPPORTED" id="basic_LTE_BANDS_SUPPORTED" class="inputDivCss border border-box item" 
		 		onblur="validateMaxAndMinVal(event)" min_value="1" max_value="62" must="1" onfocus="removeErrTip(event);"
		 		title="<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 1<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 62"/>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="basic_LTE_BANDS_SUPPORTED_err"><%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 1<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 62</div>	
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("DaiKuan")%></span>
			<select name="basic_UL_BANDWIDTH" id="basic_LTE_UL_BANDWIDTH" class="inputDivCss border border-box item">
				<option value="n25">5MHz</option>
				<option value="n50">10MHz</option>
				<option value="n75">15MHz</option>
				<option value="n100">20MHz</option>
			</select>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"> </div>
		</div>
		<div class="itemDiv pciisLock">
			<span><%=rb.getString("PinLv")%>(MHz)</span>
			<input type="text" name="basic_UL_DL_EARFCN" id="basic_LTE_UL_DL_EARFCN" class="inputDivCss border border-box item" 
					onblur="validateMaxAndMinVal(event)" min_value="0" max_value="3800" must="1" onfocus="removeErrTip(event);"
					title="<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 3800" />					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="basic_LTE_UL_DL_EARFCN_err" ><%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 3800</div>
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("ZiZhenPeiBi")%></span>
			<select name="basic_TDD_SUBFRAME_ASSIGNMENT" id="basic_LTE_TDD_SUBFRAME_ASSIGNMENT" class="inputDivCss border border-box item">
				<option value="1">1(DL:UL = 2:2)</option>
				<option value="2">2(DL:UL = 3:1)</option>
			</select>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"> </div>
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("TeShuZiZhenPeiBi")%></span>
			<select name="basic_TDD_SPECIAL_SUB_FRAME_PATTERNS" id="basic_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS" class="inputDivCss border border-box item">
				<option value="5">5</option>
				<option value="7">7</option>
			</select>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"> </div>
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("PLMN")%></span>
			<input type="text" name="basic_OAM_PLMNID" id="basic_LTE_OAM_PLMNID" class="inputDivCss border border-box item"
					onblur="validateMaxAndMinVal(event)"  min_value="10000" max_value="999999" must="1" onfocus="removeErrTip(event);"
					title="<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 10000<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 999999"/>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="basic_LTE_OAM_PLMNID_err" ><%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 10000<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 999999</div>			
		</div>
	</div>
	<div class="itemDiv" style="">
		<span><%=rb.getString("TAC")%></span>
		<input type="text" name="basic_TAC"  id="basic_TAC"class="inputDivCss border border-box item" oldValue="" value=""
				onblur="validateByRegexAndRange(event);changeTipShowOrHide(event)" min_value="0" max_value="65535" must="1" 
				vali-regex="/^(\d+\.\.){0,1}(\d+)$/" 
				title="<%=rb.getString("LiRu")%>:'12..34'<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 65535"/>
		<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
		<div class="errorTitle" id="basic_TAC_err"><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 65535</div>	
		<div class="saveTip" id="basic_TAC_tip"><%=rb.getString("DanZhanPeiZhiBuGaiBian")%></div>
	</div>
	<div class="itemDiv" style="">
		<span><%=rb.getString("JiZhanID")%></span>
		<input type="text" name="basic_CELL_IDENTITY"  id="basic_CELL_IDENTITY"class="inputDivCss border border-box item" oldValue="${TAC}" value="${TAC}"
				onblur="validateByRegexAndRange(event);changeTipShowOrHide(event)" min_value="0" max_value="268435455" must="1" 
				vali-regex="/^(\d+\.\.){0,1}(\d+)$/" 
				title="<%=rb.getString("LiRu")%>:'12..34'<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 268435455"/>
		<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
		<div class="errorTitle" id="basic_CELL_IDENTITY_err"><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 268435455</div>	
		<div class="saveTip" id="basic_CELL_IDENTITY_tip"><%=rb.getString("DanZhanPeiZhiBuGaiBian")%></div>
	</div>
	<div class="itemDiv" style="">
		<span><%=rb.getString("PCI2")%></span>
		<input type="text" name="basic_PHY_CELLID_LIST" id="basic_PHY_CELLID_LIST" class="inputDivCss border border-box item" oldValue="${PHYCELLID}" value="${PHYCELLID}"
			onblur="validateByRegexAndRange(event);changeTipShowOrHide(event)" min_value="0" max_value="503" must="1" 
			vali-regex="/^(\d+\.\.){0,1}(\d+)$/"
			title="<%=rb.getString("LiRu")%>:'12..34'<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 503"/>
		<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
		<div class="errorTitle" id="basic_PHY_CELLID_LIST_err"><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 503</div>	
		<div class="saveTip" id="basic_PHY_CELLID_LIST_tip"><%=rb.getString("DanZhanPeiZhiBuGaiBian")%></div>
	</div>
	<div class="itemDiv" style="">
		<span><%=rb.getString("GenXuLieSuoYin")%></span>
		<input type="text" name="basic_ROOT_SEQ_INDEX"  id="basic_ROOT_SEQ_INDEX" class="inputDivCss border border-box item" 
			oldValue="${ROOT_SEQUENCE_INDEX}" value="${ROOT_SEQUENCE_INDEX}"
				onblur="validateByRegexAndRange(event);changeTipShowOrHide(event)" min_value="0" max_value="837" must="1" 
				vali-regex="/^(\d+\.\.){0,1}(\d+)$/"
				title="<%=rb.getString("LiRu")%>:'12..34'<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 837"/>
		<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
		<div class="errorTitle" id="basic_ROOT_SEQ_INDEX_err"><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 837</div>	
		<div class="saveTip" id="basic_ROOT_SEQ_INDEX_tip"><%=rb.getString("DanZhanPeiZhiBuGaiBian")%></div>
	</div>
	<div class="itemDiv" style="float:none;clear:both">
		<span><%=rb.getString("HeXinWang")%></span>
		<input type="text" name="basic_SIGLINK_SERVER_LIST" id="basic_LTE_SIGLINK_SERVER_LIST" class="inputDivCss border border-box item" 
			onblur="basic_validateIPAddress1(event)" onfocus="removeErrTip(event);"/>
		<div class='titleIcon titleIcon_add' style='cursor:pointer;display: inline-block;width:26px;height:26px;vertical-align:middle;' onclick="basic_addMMEIpInputText(this)"></div>
		<div class="errorTitle" id="basic_LTE_SIGLINK_SERVER_LIST_err"><%=rb.getString("IPDiZhi")%></div>
	</div>
	<!-- IPSEC设置 -->
	<div class="omcPageTitleDiv" style="margin-bottom: 20px;" id="basic_IPSECDIV">
		<ul class="omcPageTitleContainer">
			<li class="default">IPSec <%=rb.getString("SheZhi")%></li>
		</ul>
	</div>
	<div class="IPSECSetting" id="basic_IPSECDIV">
		<div class="itemDiv">
			<span><%=rb.getString("IpsecKaiGuan")%></span>
			<select name="IPSEC_ENABLE" id="basic_PSEC_ENABLE" class="inputDivCss border border-box item">
				<option value="0"><%=rb.getString("GuanBi")%></option>
				<option value="1"><%=rb.getString("KaiQi")%></option>
			</select>
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"> </div>
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("IKEXieShangMuDiDuanKou")%></span>
			<select name="IPSEC_RIGHTIKEPORT" id="basic_IPSEC_RIGHTIKEPORT" class="inputDivCss border border-box item">
				<option value="500">500</option>
				<option value="4500">4500</option>
			</select>
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"> </div>
		</div>
		<div class="itemDiv" style="">
			<span>Left Interface</span>
			<select name="RIGHT_LEFT_INTERFACE" id="RIGHT_LEFT_INTERFACE" class="inputDivCss border border-box item">
				<option value="none">none</option>
				<option value="WAN">WAN(eth2)</option>
				<option value="PPPOE">PPPOE(pppoe-wan)</option>
			</select>				
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="basic_LEFT_INTERFACE_err"> </div>	
		</div>
		
		<div style="margin-left:40px;clear:both;height:26px;line-height:26px;width:733px;">
			<span>IPSec Tunnel <%=rb.getString("LieBiao")%></span>
			<div class='titleIcon titleIcon_add' id='tunnelAdd' style='cursor:pointer;float:right;;width:26px;height:26px;vertical-align:middle;' onclick="addTunnel(this)"></div>
		</div>
		<div style="height:130px">
			<div style="height:103px;width:733px;border:1px solid #CCE1EF;float:left;margin-right:50px;margin-left:40px">
				<table id="ipsecTunnelData"></table>
			</div>
			<div class="errorTitle" id="ipsecTunnelData_err" style="clear:both;margin-left:40px;width:700px;height:26px;"><%=rb.getString("IpSecTunnelPeiZhiGeShu")%></div>	
		</div>
		<!-- 索引1的内容 -->
		<div id="channel1" class="ipsecChannel">
			<div class="omcPageTitleDiv" style="margin-bottom: 20px;">
				<ul class="omcPageTitleContainer">
					<li class="default"></li>
				</ul>
			</div>
			<div class="itemDiv" style="display:none">
				<span><%=rb.getString("SuoYin") %></span>
				<input type="text" name="ID_1" id="ID_1" class="inputDivCss border border-box item" disabled value = '1'/>					
				<div class="errorTitle"> </div>	
			</div> 
			<div class="itemDiv">
				<span><%=rb.getString("TunnelKaiGuan")%></span>
				<select name="TUNNEL_ENABLE_1" id="basic_TUNNEL_ENABLE_1" class="inputDivCss border border-box item">
					<option value="0"><%=rb.getString("GuanBi")%></option>
					<option value="1"><%=rb.getString("KaiQi")%></option>
				</select>
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv" style="">
				<span><%=rb.getString("TunnelMingCheng")%></span>
				<input type="text" name="TUNNEL_NAME_1" id="basic_TUNNEL_NAME_1" class="inputDivCss border border-box item" 
				min_length="1" max_length="14" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="/^\w+$/" />					
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle" id="basic_TUNNEL_NAME_1_err" ><%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChangDu")%>14</div>	
			</div>
			<div class="itemDiv" style="">
				<span><%=rb.getString("TunnelWangGuan")%></span>
				<input type="text" name="TUNNEL_GATEWAY_1" id="basic_TUNNEL_GATEWAY_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);"  onfocus="removeErrTip(event);"
				js_regex="no_zh" />					
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle" id="basic_TUNNEL_GATEWAY_1_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>1-256</div>	
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("RenZhengFangShi") %></span>
				<select name="AUTHBY_1" id="basic_AUTHBY_1" class="inputDivCss border border-box item">
					<option value="psk">psk</option>
					<option value="cert">cert</option>
					<option value="aka_psk">aka_psk</option>
					<option value="aka_cert">aka_cert</option>
				</select>
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle"></div>
			</div>
			<div class="itemDiv" style="">
				<span><%=rb.getString("YuGongXiangMiYao") %></span>
				<input type="password" name="PRE_SHARED_KEY_1" id="basic_PRE_SHARED_KEY_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex = "no_zh" />
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle"  id="basic_PRE_SHARED_KEY_1_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>1-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>Left Identifier</span>
				<input type="text" name="LEFT_IDENTIFIER_1" id="basic_LEFT_IDENTIFIER_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="no_zh"/>					
				<div class="errorTitle" id="basic_LEFT_IDENTIFIER_1_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>Right Identifier</span>
				<input type="text" name="RIGHT_IDENTIFIER_1" id="basic_RIGHT_IDENTIFIER_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="no_zh"/>				
				<div class="errorTitle" id="basic_RIGHT_IDENTIFIER_1_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>leftsourceip</span>
				<input type="text" name="LEFTSOURCEIP_1" id="basic_LEFTSOURCEIP_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="/^(?:(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])|%config)$/"/>					
				<div class="errorTitle" id="basic_LEFTSOURCEIP_1_err" ><%=rb.getString("IPDiZhiHuoConfig")%></div>	
			</div>
			<div class="itemDiv">
				<span>Ike Encryption</span>
				<select name="IKE_ENCRYPTION_1" id="basic_IKE_ENCRYPTION_1" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="aes128">aes128</option>
					<option value="aes256">aes256</option>
					<option value="3des">3des</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv">
				<span>ESP Encryption</span>
				<select name="ESP_ENCRYPTION_1" id="basic_ESP_ENCRYPTION_1" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="aes128">aes128</option>
					<option value="aes256">aes256</option>
					<option value="3des">3des</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv">
				<span>Ike Dh Group</span>
				<select name="IKE_DH_GROUP_1" id="basic_IKE_DH_GROUP_1" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="modp768">modp768</option>
					<option value="modp1024">modp1024</option>
					<option value="modp1536">modp1536</option>
					<option value="modp2048">modp2048</option>
					<option value="modp3072">modp3072</option>
					<option value="modp4096">modp4096</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv">
				<span>Esp Dh Group</span>
				<select name="ESP_DH_GROUP_1" id="basic_ESP_DH_GROUP_1" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="modp768">modp768</option>
					<option value="modp1024">modp1024</option>
					<option value="modp1536">modp1536</option>
					<option value="modp2048">modp2048</option>
					<option value="modp4096">modp4096</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv">
				<span>Ike Authentication</span>
				<select name="IKE_AUTHENTICATION_1" id="basic_IKE_AUTHENTICATION_1" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="sha1">sha1</option>
					<option value="sha512">sha512</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv">
				<span>Esp Authentication</span>
				<select name="ESP_AUTHENTICATION_1" id="basic_ESP_AUTHENTICATION_1" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="sha1">sha1</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv" style="">
				<span>Keylife</span>
				<input type="text" name="KEYLIFE_1" id="basic_KEYLIFE_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="/^\d+[s|m|d]{1}$/"/>					
				<div class="errorTitle" id="basic_KEYLIFE_1_err" ><%=rb.getString("ShuZiJiaSMD")%></div>	
			</div>
			<div class="itemDiv" style="">
				<span>Ikelifetime</span>
				<input type="text" name="IKELIFETIME_1" id="basic_IKELIFETIME_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="/^\d+[s|m|d]{1}$/"/>					
				<div class="errorTitle" id="basic_IKELIFETIME_1_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
			</div>
			<div class="itemDiv" style="">
				<span>Rekeymargin</span>
				<input type="text" name="REKEYMARGIN_1" id="basic_REKEYMARGIN_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="/^\d+[s|m|d]{1}$/"/>					
				<div class="errorTitle" id="basic_REKEYMARGIN_1_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
			</div>
			<div class="itemDiv" style="">
				<span>Keyingtries</span>
				<input type="text" name="KEYINGTRIES_1" id="basic_KEYINGTRIES_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="/^(?:\d+|%forever)$/"/>					
				<div class="errorTitle" id="basic_KEYINGTRIES_1_err"><%=rb.getString("ShuZiHuoforever")%></div>	
			</div>
			<div class="itemDiv">
				<span>dpdaction</span>
				<select name="DPDACTION_1" id="basic_DPDACTION_1" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="none">none</option>
					<option value="clear">clear</option>
					<option value="hold">hold</option>
					<option value="restart">restart</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv" style="">
				<span>Dpddelay</span>
				<input type="text" name="DPDDELAY_1" id="basic_DPDDELAY_1" class="inputDivCss border border-box item"
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="/^\d+[s|m|d]{1}$/"/>				
				<div class="errorTitle" id="basic_DPDDELAY_1_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
			</div>
			<div class="itemDiv" style="">
				<span>Rootcertificates</span>
				<input type="text" name="ROOTCERTIFICATES_1" id="basic_ROOTCERTIFICATES_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="no_zh"/>					
				<div class="errorTitle" id="basic_ROOTCERTIFICATES_1_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>Certificates</span>
				<input type="text" name="CERTIFICATES_1" id="basic_CERTIFICATES_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="no_zh"/>					
				<div class="errorTitle" id="basic_CERTIFICATES_1_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>Privatekeys</span>
				<input type="text" name="PRIVATEKEYS_1" id="basic_PRIVATEKEYS_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="no_zh"/>					
				<div class="errorTitle" id="basic_PRIVATEKEYS_1_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>RightSubnet</span>
				<input type="text" name="RIGHT_SUBNET_1" id="basic_RIGHT_SUBNET_1" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="no_zh"/>				
				<div class="errorTitle" id="basic_RIGHT_SUBNET_1_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="linkbuttonGroup modifyLinkbutton" style="margin: 0 0 20px 40px;;float:left;">
			    <a id="halobConfigSub" href="#" class="linkbutton linkbutton_trend" onclick="saveTunnel1Modify('channel1')" ><span><%=rb.getString("QueDing")%></span></a>
			    <a href="#" class="linkbutton linkbutton_nowanna" onclick="cancelTunnelModify('channel1')"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
			<a href="#" class="linkbutton viewLinkbutton" style="margin-left:40px;display:none;" onclick="cancelTunnelModify('channel1')"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
		<!-- 索引2的内容 -->
		<div id="channel2" class="ipsecChannel">
			<div class="omcPageTitleDiv" style="margin-bottom: 20px;">
				<ul class="omcPageTitleContainer">
					<li class="default"></li>
				</ul>
			</div>
			<div class="itemDiv" style="display:none">
				<span><%=rb.getString("SuoYin") %></span>
				<input type="text" name="ID_2" id="ID_2" class="inputDivCss border border-box item" disabled value = '2'/>					
				<div class="errorTitle"> </div>	
			</div> 
			<div class="itemDiv">
				<span><%=rb.getString("TunnelKaiGuan")%></span>
				<select name="TUNNEL_ENABLE_2" id="basic_TUNNEL_ENABLE_2" class="inputDivCss border border-box item">
					<option value="0"><%=rb.getString("GuanBi")%></option>
					<option value="1"><%=rb.getString("KaiQi")%></option>
				</select>
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv" style="">
				<span><%=rb.getString("TunnelMingCheng")%></span>
				<input type="text" name="TUNNEL_NAME_2" id="basic_TUNNEL_NAME_2" class="inputDivCss border border-box item" 
				min_length="1" max_length="14" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="/^\w+$/" />					
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle" id="basic_TUNNEL_NAME_2_err" ><%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChangDu")%>14</div>	
			</div>
			<div class="itemDiv" style="">
				<span><%=rb.getString("TunnelWangGuan")%></span>
				<input type="text" name="TUNNEL_GATEWAY_2" id="basic_TUNNEL_GATEWAY_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);"  onfocus="removeErrTip(event);"
				js_regex="no_zh" />					
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle" id="basic_TUNNEL_GATEWAY_2_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>1-256</div>	
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("RenZhengFangShi") %></span>
				<select name="AUTHBY_2" id="basic_AUTHBY_2" class="inputDivCss border border-box item">
					<option value="psk">psk</option>
					<option value="cert">cert</option>
					<option value="aka_psk">aka_psk</option>
					<option value="aka_cert">aka_cert</option>
				</select>
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle"></div>
			</div>
			<div class="itemDiv" style="">
				<span><%=rb.getString("YuGongXiangMiYao") %></span>
				<input type="password" name="PRE_SHARED_KEY_2" id="basic_PRE_SHARED_KEY_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex = "no_zh" />
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle"  id="basic_PRE_SHARED_KEY_2_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>1-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>Left Identifier</span>
				<input type="text" name="LEFT_IDENTIFIER_2" id="basic_LEFT_IDENTIFIER_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="no_zh"/>					
				<div class="errorTitle" id="basic_LEFT_IDENTIFIER_2_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>Right Identifier</span>
				<input type="text" name="RIGHT_IDENTIFIER_2" id="basic_RIGHT_IDENTIFIER_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="no_zh"/>				
				<div class="errorTitle" id="basic_RIGHT_IDENTIFIER_2_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>leftsourceip</span>
				<input type="text" name="LEFTSOURCEIP_2" id="basic_LEFTSOURCEIP_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="/^(?:(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])|%config)$/"/>					
				<div class="errorTitle" id="basic_LEFTSOURCEIP_2_err" ><%=rb.getString("IPDiZhiHuoConfig")%></div>	
			</div>
			<div class="itemDiv">
				<span>Ike Encryption</span>
				<select name="IKE_ENCRYPTION_2" id="basic_IKE_ENCRYPTION_2" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="aes128">aes128</option>
					<option value="aes256">aes256</option>
					<option value="3des">3des</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv">
				<span>ESP Encryption</span>
				<select name="ESP_ENCRYPTION_2" id="basic_ESP_ENCRYPTION_2" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="aes128">aes128</option>
					<option value="aes256">aes256</option>
					<option value="3des">3des</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv">
				<span>Ike Dh Group</span>
				<select name="IKE_DH_GROUP_2" id="basic_IKE_DH_GROUP_2" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="modp768">modp768</option>
					<option value="modp1024">modp1024</option>
					<option value="modp1536">modp1536</option>
					<option value="modp2048">modp2048</option>
					<option value="modp3072">modp3072</option>
					<option value="modp4096">modp4096</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv">
				<span>Esp Dh Group</span>
				<select name="ESP_DH_GROUP_2" id="basic_ESP_DH_GROUP_2" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="modp768">modp768</option>
					<option value="modp1024">modp1024</option>
					<option value="modp1536">modp1536</option>
					<option value="modp2048">modp2048</option>
					<option value="modp4096">modp4096</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv">
				<span>Ike Authentication</span>
				<select name="IKE_AUTHENTICATION_2" id="basic_IKE_AUTHENTICATION_2" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="sha1">sha1</option>
					<option value="sha512">sha512</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv">
				<span>Esp Authentication</span>
				<select name="ESP_AUTHENTICATION_2" id="basic_ESP_AUTHENTICATION_2" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="sha1">sha1</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv" style="">
				<span>Keylife</span>
				<input type="text" name="KEYLIFE_2" id="basic_KEYLIFE_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="/^\d+[s|m|d]{1}$/"/>					
				<div class="errorTitle" id="basic_KEYLIFE_2_err" ><%=rb.getString("ShuZiJiaSMD")%></div>	
			</div>
			<div class="itemDiv" style="">
				<span>Ikelifetime</span>
				<input type="text" name="IKELIFETIME_2" id="basic_IKELIFETIME_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="/^\d+[s|m|d]{1}$/"/>					
				<div class="errorTitle" id="basic_IKELIFETIME_2_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
			</div>
			<div class="itemDiv" style="">
				<span>Rekeymargin</span>
				<input type="text" name="REKEYMARGIN_2" id="basic_REKEYMARGIN_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
				js_regex="/^\d+[s|m|d]{1}$/"/>					
				<div class="errorTitle" id="basic_REKEYMARGIN_2_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
			</div>
			<div class="itemDiv" style="">
				<span>Keyingtries</span>
				<input type="text" name="KEYINGTRIES_2" id="basic_KEYINGTRIES_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="/^(?:\d+|%forever)$/"/>					
				<div class="errorTitle" id="basic_KEYINGTRIES_2_err"><%=rb.getString("ShuZiHuoforever")%></div>	
			</div>
			<div class="itemDiv">
				<span>dpdaction</span>
				<select name="DPDACTION_2" id="basic_DPDACTION_2" class="inputDivCss border border-box item">
					<option value="">null</option>
					<option value="none">none</option>
					<option value="clear">clear</option>
					<option value="hold">hold</option>
					<option value="restart">restart</option>
				</select>
				<div class="errorTitle"> </div>
			</div>
			<div class="itemDiv" style="">
				<span>Dpddelay</span>
				<input type="text" name="DPDDELAY_2" id="basic_DPDDELAY_2" class="inputDivCss border border-box item"
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="/^\d+[s|m|d]{1}$/"/>				
				<div class="errorTitle" id="basic_DPDDELAY_2_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
			</div>
			<div class="itemDiv" style="">
				<span>Rootcertificates</span>
				<input type="text" name="ROOTCERTIFICATES_2" id="basic_ROOTCERTIFICATES_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="no_zh"/>					
				<div class="errorTitle" id="basic_ROOTCERTIFICATES_2_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>Certificates</span>
				<input type="text" name="CERTIFICATES_2" id="basic_CERTIFICATES_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="no_zh"/>					
				<div class="errorTitle" id="basic_CERTIFICATES_2_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>Privatekeys</span>
				<input type="text" name="PRIVATEKEYS_2" id="basic_PRIVATEKEYS_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="no_zh"/>					
				<div class="errorTitle" id="basic_PRIVATEKEYS_2_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="itemDiv" style="">
				<span>RightSubnet</span>
				<input type="text" name="RIGHT_SUBNET_2" id="basic_RIGHT_SUBNET_2" class="inputDivCss border border-box item" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
				js_regex="no_zh"/>				
				<div class="errorTitle" id="basic_RIGHT_SUBNET_2_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
			</div>
			<div class="linkbuttonGroup modifyLinkbutton" style="margin: 0 0 20px 40px;;float:left;">
			    <a id="halobConfigSub" href="#" class="linkbutton linkbutton_trend" onclick="saveTunnel2Modify('channel2')" ><span><%=rb.getString("QueDing")%></span></a>
			    <a href="#" class="linkbutton linkbutton_nowanna" onclick="cancelTunnelModify('channel2')"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
			<a href="#" class="linkbutton viewLinkbutton" style="margin-left:40px;display:none;" onclick="cancelTunnelModify('channel2')"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
		
		<div class="linkbuttonGroup" id="buttonGroupView" style="margin: 40px 0 20px 40px;float:left;">
		    <a id="halobConfigSub" href="#" class="linkbutton linkbutton_trend" onclick="commonConfigSubmit()" ><span><%=rb.getString("QueDing")%></span></a>
		    <a href="#" class="linkbutton linkbutton_nowanna" onclick="closeCommonConfig()"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
		<a class="selfConfigSucTip" style="display:none;margin-top: 40px;margin-bottom: 20px;float:left;"><%=rb.getString("JiChuPeiZhiXiuGaiWanCheng")%></a>
	</div>
</div>
<script type="text/javascript">
var channelNumStr = "";
var dataList = [];
var mmeNum = '';
$(function() {
	var cellIpsecInfos = "";
	// config_type 1-基础配置，2-halob基础配置
	closeLoading();
	//如果SAS开关打开，则该参数不可配置
	if(SASEnble == "1"){
		$("#basic_LTE_UL_BANDWIDTH,#basic_LTE_UL_DL_EARFCN").prop("disabled",true);
	}
	var selectedRow = $("#publicConfig").datagrid("getSelected");
	$("#ipsecTunnelData").datagrid({
		border:false,
    	fit:true,
        singleSelect:true,
        rownumbers:false,
        fitColumns: false,
        striped:true,
        singleSelect:true,
        columns: [[
                   {field: 'tdId',hidden:true},
                   {field: 'id',width: 100, title: 'Tunnel ID'},
                   {field: 'zhangtai',width: 100, title: '<%=rb.getString("TunnelKaiGuan")%>',formatter : tunnelSwitchStr},
                   {field: 'mingcheng',width: 165,title: '<%=rb.getString("TunnelMingCheng")%>'},
                   {field: 'wangguan',width: 165, title: '<%=rb.getString("TunnelWangGuan")%>'},
                   {field: 'fangshi',width: 103, title: '<%=rb.getString("RenZhengFangShi")%>'},
                   {field: 'operate',fixed:true,fixed:true,width: 100,title:'<%=rb.getString("CaoZuo")%>',formatter: downOrDelExportFun},
                   
        ]],
        data : dataList
	})
	var params={};
	params.id = selectedRow.id;
	params.config_type = "1";
	$.post("${ctx}/cell/halobSelfConfig/getCurrConfigParamContent.action", params, function(data){
		if(data){
			dataList = [];
			$("input[name='basic_BANDS_SUPPORTED']").val(data.bands_support);
			$("select[name='basic_UL_BANDWIDTH']").val(data.band_width);
			$("input[name='basic_UL_DL_EARFCN']").val(data.frequency);
			$("select[name='basic_TDD_SUBFRAME_ASSIGNMENT']").val(data.subframe_assignment);
			$("select[name='basic_TDD_SPECIAL_SUB_FRAME_PATTERNS']").val(data.special_subframe_patterns);
			$("input[name='basic_OAM_PLMNID']").val(data.plmn_id);
			$("input[name='basic_TAC']").val(data.tac).attr('oldvalue',data.tac);
			$("input[name='basic_CELL_IDENTITY']").val(data.cell_identity).attr('oldvalue',data.cell_identity);
			$("input[name='basic_PHY_CELLID_LIST']").val(data.phycellid).attr('oldvalue',data.phycellid);
			$("input[name='basic_ROOT_SEQ_INDEX']").val(data.root_sequence_index).attr('oldvalue',data.root_sequence_index);
			var mmeArr = data.mme_ip;
			if(data.mme_ip){
				mmeArr = data.mme_ip.split(",");
			}
			//当mme地址为n且n>1时，需要重新生成n-1个输入框
			if (mmeArr && mmeArr.length > 1) {
				//初始化第一个输入框的值和oldValue值
				var firstMMEDiv = $("input[name='basic_SIGLINK_SERVER_LIST']:first");
				firstMMEDiv.val(mmeArr[0]);
				firstMMEDiv.attr("oldValue", mmeArr[0]);
				
				var newMMEInputCount = mmeArr.length;
				//生成n-1个输入框
				for(var count = 1; count < newMMEInputCount; count++) {
					mmeNum = count;
					//添加新的输入框
					var $mmeSpan = $("<span style='display:inline-block;visibility:hidden'><%=rb.getString("HeXinWang")%></span>");
					var $mmeDiv = $("<div class='itemDiv'></div>");
					var $mmeInput = $("<input type='text' name='basic_SIGLINK_SERVER_LIST' id='basic_SIGLINK_SERVER_LIST_"+count+"' class='inputDivCss border border-box item' "
						 + "onblur='validateIPAddress(event)' onfocus='removeErrTip(event);' oldValue='" + mmeArr[count] + "' value='" + mmeArr[count] + "' style='margin-left:0px'></input>");
					var $mmeButtn = $("<div class='titleIcon titleIcon_sub' style='cursor:pointer;display: inline-block;width:26px;height:26px;vertical-align:middle;margin-left:5px;' onclick='basic_removeMMEIpInputText(this)'></div>" 
							+"<div class='errorTitle' id='basic_SIGLINK_SERVER_LIST_" + count + "_err'><%=rb.getString("IPDiZhi")%></div>");
					
					$mmeDiv.append($mmeSpan);
					$mmeDiv.append($mmeInput);
					$mmeDiv.append($mmeButtn);
					$mmeDiv.addClass("clearBoth");
					
					//将新增的输入框插入到plmn参数之前
					var plmnidDiv = $("#basic_IPSECDIV");
					$mmeDiv.insertBefore(plmnidDiv);
				}
			} else {
				var firstMMEDiv = $("input[name='basic_SIGLINK_SERVER_LIST']:first");
				firstMMEDiv.val(mmeArr);
				firstMMEDiv.attr("oldValue", mmeArr);
			}
			$("select[name='IPSEC_ENABLE']").val(data.ipsec_enable);
			//如果ipsec开关关闭打开不影响任何东西
			$("select[name='IPSEC_RIGHTIKEPORT']").val(data.ipsec_rightikeport);
			$("select[name='RIGHT_LEFT_INTERFACE']").val(data.left_interface);
			cellIpsecInfos = data.ipsecList;
			if(cellIpsecInfos.length== 1){
				$("#channel2").hide();
			}
			if(cellIpsecInfos.length > 0 ){
				for(var i=0;i<cellIpsecInfos.length;i++){
					if(i==0){
						channelNumStr = "1";
						var dataListTunnel1={};
						var ID = cellIpsecInfos[i].ipsec_index;
						dataListTunnel1.tdId = 'channel'+1;
						dataListTunnel1.id = 1;
						dataListTunnel1.mingcheng = cellIpsecInfos[i].tunnel_name;
						dataListTunnel1.wangguan = cellIpsecInfos[i].tunnel_gateway;
						dataListTunnel1.fangshi = cellIpsecInfos[i].authby;
						dataListTunnel1.zhangtai = cellIpsecInfos[i].tunnel_enable;
						dataList.push(dataListTunnel1);
					}
					if(i==1){
						channelNumStr = "2";
						var ID = cellIpsecInfos[i].ipsec_index;
						var dataListTunnel2={};
						var ID = cellIpsecInfos[i].ipsec_index;
						dataListTunnel2.tdId = 'channel'+2;
						dataListTunnel2.id = 2;
						dataListTunnel2.mingcheng = cellIpsecInfos[i].tunnel_name;
						dataListTunnel2.wangguan = cellIpsecInfos[i].tunnel_gateway;
						dataListTunnel2.fangshi = cellIpsecInfos[i].authby;
						dataListTunnel2.zhangtai = cellIpsecInfos[i].tunnel_enable;
						dataList.push(dataListTunnel2);
						$("#tunnelAdd").removeClass('titleIcon_add').addClass('titleIcon_add_disabled');
					}
					//创建ID文本框
					$("input[name='ID_"+(i+1)+"']").val(ID) ;
					$("select[name='TUNNEL_ENABLE_"+(i+1)+"']").val(cellIpsecInfos[i].tunnel_enable).attr('oldValue',cellIpsecInfos[i].tunnel_enable);
					$("input[name='TUNNEL_NAME_"+(i+1)+"']").val(cellIpsecInfos[i].tunnel_name).attr('oldValue',cellIpsecInfos[i].tunnel_name);
					$("input[name='TUNNEL_GATEWAY_"+(i+1)+"']").val(cellIpsecInfos[i].tunnel_gateway).attr('oldValue',cellIpsecInfos[i].tunnel_gateway);
					$("input[name='LEFT_IDENTIFIER_"+(i+1)+"']").val(cellIpsecInfos[i].left_identifier).attr('oldValue',cellIpsecInfos[i].tunnel_gateway);
					$("input[name='RIGHT_IDENTIFIER_"+(i+1)+"']").val(cellIpsecInfos[i].right_identifier).attr('oldValue',cellIpsecInfos[i].right_identifier);
					$("select[name='AUTHBY_"+(i+1)+"']").val(cellIpsecInfos[i].authby).attr('oldValue',cellIpsecInfos[i].authby);
					$("input[name='PRE_SHARED_KEY_"+(i+1)+"']").val(cellIpsecInfos[i].pre_shared_key).attr('oldValue',cellIpsecInfos[i].pre_shared_key) ;
					$("input[name='LEFTSOURCEIP_"+(i+1)+"']").val(cellIpsecInfos[i].leftsourceip).attr('oldValue',cellIpsecInfos[i].leftsourceip) ;
					$("select[name='IKE_ENCRYPTION_"+(i+1)+"']").val(cellIpsecInfos[i].ike_encryption).attr('oldValue',cellIpsecInfos[i].ike_encryption)  ;
					$("select[name='ESP_ENCRYPTION_"+(i+1)+"']").val(cellIpsecInfos[i].esp_encryption).attr('oldValue',cellIpsecInfos[i].esp_encryption) ;
					$("select[name='IKE_DH_GROUP_"+(i+1)+"']").val(cellIpsecInfos[i].ike_dh_group).attr('oldValue',cellIpsecInfos[i].ike_dh_group) ;
					$("select[name='ESP_DH_GROUP_"+(i+1)+"']").val(cellIpsecInfos[i].esp_dh_group).attr('oldValue',cellIpsecInfos[i].esp_dh_group) ;
					$("select[name='IKE_AUTHENTICATION_"+(i+1)+"']").val(cellIpsecInfos[i].ike_authentication).attr('oldValue',cellIpsecInfos[i].ike_authentication) ;
					$("select[name='ESP_AUTHENTICATION_"+(i+1)+"']").val(cellIpsecInfos[i].esp_authentication).attr('oldValue',cellIpsecInfos[i].esp_authentication) ;
					$("input[name='KEYLIFE_"+(i+1)+"']").val(cellIpsecInfos[i].keylife).attr('oldValue',cellIpsecInfos[i].keylife);
					$("input[name='IKELIFETIME_"+(i+1)+"']").val(cellIpsecInfos[i].ikelifetime).attr('oldValue',cellIpsecInfos[i].ikelifetime) ;
					$("input[name='REKEYMARGIN_"+(i+1)+"']").val(cellIpsecInfos[i].rekeymargin).attr('oldValue',cellIpsecInfos[i].rekeymargin) ;
					$("input[name='KEYINGTRIES_"+(i+1)+"']").val(cellIpsecInfos[i].keyingtries).attr('oldValue',cellIpsecInfos[i].keyingtries) ;
					$("select[name='DPDACTION_"+(i+1)+"']").val(cellIpsecInfos[i].dpdaction).attr('oldValue',cellIpsecInfos[i].dpdaction) ;
					$("input[name='DPDDELAY_"+(i+1)+"']").val(cellIpsecInfos[i].dpddelay).attr('oldValue',cellIpsecInfos[i].dpddelay) ;
					$("input[name='ROOTCERTIFICATES_"+(i+1)+"']").val(cellIpsecInfos[i].rootcertificates).attr('oldValue',cellIpsecInfos[i].rootcertificates) ;
					$("input[name='CERTIFICATES_"+(i+1)+"']").val(cellIpsecInfos[i].certificates).attr('oldValue',cellIpsecInfos[i].certificates) ;
					$("input[name='PRIVATEKEYS_"+(i+1)+"']").val(cellIpsecInfos[i].privatekeys).attr('oldValue',cellIpsecInfos[i].privatekeys) ;
					$("input[name='RIGHT_SUBNET_"+(i+1)+"']").val(cellIpsecInfos[i].right_subnet).attr('oldValue',cellIpsecInfos[i].right_subnet) ;
				}
				$("#ipsecTunnelData").datagrid({data : dataList});
			}
		}
		if(isView == "true"){
			$(".wirelessSetting input,select").prop("disabled",true);
			$(".titleIcon_add,.titleIcon_sub,.titleIcon_add_disabled").hide();
			$("#buttonGroupView").hide();
			$(".modifyLinkbutton").hide();
			$(".viewLinkbutton").show();
		}
	},'json')
})

//添加一个MME地址输入框
function basic_addMMEIpInputText(e) {
	mmeNum +=1;
	//添加新的输入框
	var oldValue =$("input[id='basic_SIGLINK_SERVER_LIST']").first().attr("oldValue");
	var $mmeSpan = $("<span style='display:inline-block;visibility:hidden'><%=rb.getString("HeXinWang")%></span>");
	var $mmeDiv = $("<div class='itemDiv'></div>");
	var $mmeInput = $("<input type='text' name='basic_SIGLINK_SERVER_LIST' id='basic_SIGLINK_SERVER_LIST_"+mmeNum+ "' class='inputDivCss border border-box item' "
		 + "id='"+Math.random()+"' onblur='basic_validateIPAddress11(event)'  oldValue='"+oldValue+"' style='margin-left:0px'></input>");
	var $mmeButtn = $("<div class='titleIcon titleIcon_sub' style='cursor:pointer;display: inline-block;width:26px;height:26px;vertical-align:middle;margin-left:5px;' onclick='basic_removeMMEIpInputText(this)'></div>"
		+"<div class='errorTitle' id='basic_SIGLINK_SERVER_LIST_" + mmeNum + "_err'><%=rb.getString("IPDiZhi")%></div>");
	
	$mmeDiv.append($mmeSpan);
	$mmeDiv.append($mmeInput);
	$mmeDiv.append($mmeButtn);
	$mmeDiv.addClass("clearBoth");
	//将新增的输入框插入到plmn参数之前
	var plmnidDiv = $("#basic_IPSECDIV");
	$mmeDiv.insertBefore(plmnidDiv); 
	
	//将当前输入框后面的图标改为删除图标，并重新绑定事件
	$(e).children("img").attr("src","${ctx}/css/images/bi/setting_add.png");
	$(e).attr("onclick", "basic_addMMEIpInputText(this)");
}
//删除选中的MME地址输入框
function basic_removeMMEIpInputText(e) {
	$(e).parent("div").remove();
	//如果当前第一个MME输入框的span不显示，将其显示
	var mmeDiv = $("input[name='basic_SIGLINK_SERVER_LIST']:first");
	var isVisible = mmeDiv.parent("div").children("span").css("visibility");
	if (isVisible == "hidden") {
		mmeDiv.parent("div").children("span").css("visibility","visible");
	}
}

function basic_authbyChange(e){
	var ele = $(e["target"]);
	var name = ele.attr("name");
	var value = ele.val();
	var oldValue = ele.nextAll("input").eq(0).val();
	var type = /^[^\d]*(\d+)$/;
	var index = type.exec(name)[1];
	if(oldValue == "cert" || oldValue == "aka_cert"){
		if(oldValue == "cert" && value == "aka_cert"){
			$.messager.alert(TiShi, "<%=rb.getString("BuNnengGengGai")%>");
			ele.val(oldValue);
		}else if(oldValue == "aka_cert" && value == "cert"){
			$.messager.alert(TiShi, "<%=rb.getString("BuNnengGengGai")%>");
			ele.val(oldValue);
		}
	}else{
		if(value == "cert" || value == "aka_cert"){
			$.messager.alert(TiShi, "<%=rb.getString("BuNnengGengGai")%>");
			ele.val(oldValue);
		}
	}
	basic_changeColor(ele.attr("id"));
}
function basic_validateMaxAndMinVal1(e) {
    validateMaxAndMinVal(e);
}
function basic_validateIPAddress1(e){
  	var ele = $(e["target"]);
    validateIPAddress(e);
}
function basic_validateIPAddress11(e){
  	var ele = $(e["target"]);
    validateIPAddress(e);
}
function basic_bindInterface(e){
	var ele = $(e["target"]);
	basic_changeColor(ele.attr("id"))
}
function basic_changeColor(id){
	var opts = $("#"+id).find("option");
	//需要对OLD的命名规则为basic_OLD_mibdn,此时发过来的id为basic_mibdn，需要转换
	var i = id.indexOf("_");
	var prefix = id.substring(0,i+1);
	var suffix = id.substring(i,id.length)
	var OLD_ID = prefix+"OLD"+suffix;
	var oldValue=$("#"+OLD_ID).val();
	
	var nowValue = $("#"+id).val();
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == nowValue){
			$(opts[i]).attr("selected",true);
			if(oldValue!=nowValue){
				$("#"+id).css({"color":"blue"});
				$("#"+id+" option").css({"color":"black"});
			}else{
				$("#"+id).css({"color":"black"});	
			}
			break;
		}
	}
}

function commonConfigSubmit(){
	var isPass = true;
	var allInput = $("#basicCommonConfig input");
	for(var i=0;i<allInput.length;i++){
		if($(allInput[i]).val()==""){
			$(allInput[i]).addClass('err_border');
		};
	}
	if ($("#commonConfigViewOrModify .item.err_border").length > 0) {
		var positionTop =parseInt($($("#commonConfigViewOrModify .item.err_border")[0]).offset().top-0);
 		$(".wirelessSetting").animate({scrollTop:positionTop},200);
	 	isPass = false;
	}
	if(dataList.length == 0){
		$("#ipsecTunnelData_err").show();
	 	isPass = false;
	}
	if(!isPass){
		return;
	}
	var paramMap = {};
	paramMap.LTE_BANDS_SUPPORTED = $("#basic_LTE_BANDS_SUPPORTED").val() ;
	paramMap.LTE_DL_BANDWIDTH = $("select[name='basic_UL_BANDWIDTH']").val();
	paramMap.LTE_DL_EARFCN = $("#basic_LTE_UL_DL_EARFCN").val() ;
	paramMap.LTE_TDD_SUBFRAME_ASSIGNMENT = $("select[name='basic_TDD_SUBFRAME_ASSIGNMENT']").val();
	paramMap.LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS = $("select[name='basic_TDD_SPECIAL_SUB_FRAME_PATTERNS']").val();
	paramMap.LTE_OAM_PLMNID = $("#basic_LTE_OAM_PLMNID").val() ;
	paramMap.LTE_TAC = $("#basic_TAC").val() ;
	paramMap.LTE_CELL_IDENTITY = $("#basic_CELL_IDENTITY").val() ;
	paramMap.LTE_PHY_CELLID_LIST = $("#basic_PHY_CELLID_LIST").val() ;
	paramMap.LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST = $("#basic_ROOT_SEQ_INDEX").val() ;
	
	var mmeSrt = $("input[name='basic_SIGLINK_SERVER_LIST']");
	var mmeList = [];
	$.each(mmeSrt,function(index,item){
		mmeList.push($(item).val());
	});
	paramMap.LTE_SIGLINK_SERVER_LIST = mmeList.join(",");
	paramMap.IPSEC_ENABLE = $("select[name='IPSEC_ENABLE']").val();
	paramMap.IPSEC_RIGHTIKEPORT = $("select[name='IPSEC_RIGHTIKEPORT']").val();
	paramMap.LEFT_INTERFACE = $("select[name='RIGHT_LEFT_INTERFACE']").val();
	paramMap=JSON.stringify(paramMap);
	
	var ipsecList = [];
	$.each(dataList,function(index,ele){
		var ipsecEle = {};
		if(ele.tdId == 'channel1'){
			showChannel1 = true;
			i = 1;
			ipsecEle["ipsec_index"]= ele.id;
			ipsecEle["TUNNEL_ENABLE"]=$("select[name='TUNNEL_ENABLE_"+i+"']").val();
			ipsecEle["TUNNEL_ENABLE"]=$("select[name='TUNNEL_ENABLE_"+i+"']").val();
			ipsecEle["TUNNEL_NAME"]=$("input[name='TUNNEL_NAME_"+i+"']").val();
			ipsecEle["TUNNEL_GATEWAY"]=$("input[name='TUNNEL_GATEWAY_"+i+"']").val();
			ipsecEle["LEFT_IDENTIFIER"]=$("input[name='LEFT_IDENTIFIER_"+i+"']").val();
			ipsecEle["RIGHT_IDENTIFIER"]=$("input[name='RIGHT_IDENTIFIER_"+i+"']").val();
			ipsecEle["AUTHBY"]=$("select[name='AUTHBY_"+i+"']").val();
			ipsecEle["PRE_SHARED_KEY"]=$("input[name='PRE_SHARED_KEY_"+i+"']").val() ;
			ipsecEle["LEFTSOURCEIP"]=$("input[name='LEFTSOURCEIP_"+i+"']").val() ;
			ipsecEle["IKE_ENCRYPTION"]=$("select[name='IKE_ENCRYPTION_"+i+"']").val() ;
			ipsecEle["ESP_ENCRYPTION"]=$("select[name='ESP_ENCRYPTION_"+i+"']").val() ;
			ipsecEle["IKE_DH_GROUP"]=$("select[name='IKE_DH_GROUP_"+i+"']").val() ;
			ipsecEle["ESP_DH_GROUP"]=$("select[name='ESP_DH_GROUP_"+i+"']").val() ;
			ipsecEle["IKE_AUTHENTICATION"]=$("select[name='IKE_AUTHENTICATION_"+i+"']").val() ;
			ipsecEle["ESP_AUTHENTICATION"]=$("select[name='ESP_AUTHENTICATION_"+i+"']").val() ;
			ipsecEle["KEYLIFE"]=$("input[name='KEYLIFE_"+i+"']").val() ;
			ipsecEle["IKELIFETIME"]=$("input[name='IKELIFETIME_"+i+"']").val() ;
			ipsecEle["REKEYMARGIN"]=$("input[name='REKEYMARGIN_"+i+"']").val() ;
			ipsecEle["KEYINGTRIES"]=$("input[name='KEYINGTRIES_"+i+"']").val() ;
			ipsecEle["DPDACTION"]=$("select[name='DPDACTION_"+i+"']").val() ;
			ipsecEle["DPDDELAY"]=$("input[name='DPDDELAY_"+i+"']").val() ;
			ipsecEle["ROOTCERTIFICATES"]=$("input[name='ROOTCERTIFICATES_"+i+"']").val() ;
			ipsecEle["CERTIFICATES"]=$("input[name='CERTIFICATES_"+i+"']").val() ;
			ipsecEle["PRIVATEKEYS"]=$("input[name='PRIVATEKEYS_"+i+"']").val() ;
			ipsecEle["RIGHT_SUBNET"]=$("input[name='RIGHT_SUBNET_"+i+"']").val() ;
			ipsecEle["LEFT_INTERFACE"]=$("input[name='LEFT_INTERFACE_"+i+"']").val() ;
			ipsecList.push(ipsecEle);
		}
		if(ele.tdId == 'channel2'){
			i = 2;
			ipsecEle["ipsec_index"]= ele.id;
			ipsecEle["TUNNEL_ENABLE"]=$("select[name='TUNNEL_ENABLE_"+i+"']").val();
			ipsecEle["TUNNEL_NAME"]=$("input[name='TUNNEL_NAME_"+i+"']").val();
			ipsecEle["TUNNEL_GATEWAY"]=$("input[name='TUNNEL_GATEWAY_"+i+"']").val();
			ipsecEle["LEFT_IDENTIFIER"]=$("input[name='LEFT_IDENTIFIER_"+i+"']").val();
			ipsecEle["RIGHT_IDENTIFIER"]=$("input[name='RIGHT_IDENTIFIER_"+i+"']").val();
			ipsecEle["AUTHBY"]=$("select[name='AUTHBY_"+i+"']").val();
			ipsecEle["PRE_SHARED_KEY"]=$("input[name='PRE_SHARED_KEY_"+i+"']").val() ;
			ipsecEle["LEFTSOURCEIP"]=$("input[name='LEFTSOURCEIP_"+i+"']").val() ;
			ipsecEle["IKE_ENCRYPTION"]=$("select[name='IKE_ENCRYPTION_"+i+"']").val() ;
			ipsecEle["ESP_ENCRYPTION"]=$("select[name='ESP_ENCRYPTION_"+i+"']").val() ;
			ipsecEle["IKE_DH_GROUP"]=$("select[name='IKE_DH_GROUP_"+i+"']").val() ;
			ipsecEle["ESP_DH_GROUP"]=$("select[name='ESP_DH_GROUP_"+i+"']").val() ;
			ipsecEle["IKE_AUTHENTICATION"]=$("select[name='IKE_AUTHENTICATION_"+i+"']").val() ;
			ipsecEle["ESP_AUTHENTICATION"]=$("select[name='ESP_AUTHENTICATION_"+i+"']").val() ;
			ipsecEle["KEYLIFE"]=$("input[name='KEYLIFE_"+i+"']").val() ;
			ipsecEle["IKELIFETIME"]=$("input[name='IKELIFETIME_"+i+"']").val() ;
			ipsecEle["REKEYMARGIN"]=$("input[name='REKEYMARGIN_"+i+"']").val() ;
			ipsecEle["KEYINGTRIES"]=$("input[name='KEYINGTRIES_"+i+"']").val() ;
			ipsecEle["DPDACTION"]=$("select[name='DPDACTION_"+i+"']").val() ;
			ipsecEle["DPDDELAY"]=$("input[name='DPDDELAY_"+i+"']").val() ;
			ipsecEle["ROOTCERTIFICATES"]=$("input[name='ROOTCERTIFICATES_"+i+"']").val() ;
			ipsecEle["CERTIFICATES"]=$("input[name='CERTIFICATES_"+i+"']").val() ;
			ipsecEle["PRIVATEKEYS"]=$("input[name='PRIVATEKEYS_"+i+"']").val() ;
			ipsecEle["RIGHT_SUBNET"]=$("input[name='RIGHT_SUBNET_"+i+"']").val() ;
			ipsecEle["LEFT_INTERFACE"]=$("input[name='LEFT_INTERFACE_"+i+"']").val() ;
			ipsecList.push(ipsecEle);
		}
	})
	ipsecList=JSON.stringify(ipsecList); 
	var params={};
	params.paramMap = paramMap;
	params.ipsecList = ipsecList;
	params.config_type = "1";
	$.post("${ctx}/cell/halobSelfConfig/saveAllConfigParamValue.action", params, function(data){
		if(data.success){
			$(".selfConfigSucTip").show();
			setTimeout('$(".selfConfigSucTip").fadeOut()',1000);
			$("#publicConfig").datagrid("reload");
			setTimeout('closeCommonConfig()',1000);
		}else{
			$.messager.alert("<%=rb.getString("TiShi")%>", data.message);
      	    return;
		}
	},"json");
}

function specialChar(e){
	 var ele = $(e["target"]);
	
	//如果在之前的操作中已经报错，则不再进行这一步判断
	if($("#"+ele.attr("id")+"_err").is(":visible")){
		return ;
	}
	var mustVal = ele.attr("must") == undefined ? "" : ele.attr("must");
	//如果不是必填项，且输入的值为空，则不做下面的验证
	if(mustVal != "1" && ele.val() == ""){
		return ;
	}
	var js_regex;
	if(ele.attr("js_regex") == "no_zh"){
		/*  验证不允许输入中文，由于下面的正则表达式在用eval转换时出错，因此做特殊处理 **/
		js_regex = /^(?:(?![\u4E00-\u9FA5]|[\uFE30-\uFFA0]).)+$/;
	}else{
		js_regex = eval("(" + ele.attr("js_regex") + ")");
	}
	
	var val = ele.val();
	if(js_regex.test(val)){
		$("#" + ele.attr("id") + "_err").hide();
    	ele.removeClass("err_border");
	}else{
		$("#" + ele.attr("id") + "_err").show();
		ele.addClass("err_border");
	}
}
function removeErrTip(e){
	/* var ele = $(e["target"]);
	//如果在之前的操作中已经报错，则不再进行这一步判断
	if($("#"+ele.attr("id")+"_err").is(":visible")){
		ele.removeClass("err_border");
		$("#" + ele.attr("id") + "_err").hide();
	} */
}
var XiuGai = '<%=rb.getString("XiuGai")%>';
//操作列
function downOrDelExportFun(value, rowData, rowIndex){
    var gridData = JSON.stringify(rowData);
	value = "";
	if(isView == 'true'){
		value += "<div class='operationDiv operation_view' title='"+ChaKan+"' onclick='viewChannel(" + gridData + ")'></div>";
	}else{
	    value +=  "<div class='operationDiv operation_edit' title='"+XiuGai+"' style='margin-left:15px;cursor:pointer;' onclick='modifyChannel(" + gridData + ")'></div>";
	    value +=  "<div class='operationDiv operation_delete' title='"+ShanChu+"' style='margin-left:15px;cursor:pointer;' onclick='deleteChannel(" + gridData + ")'></div>";
	}
	return value;
}
function viewChannel(rowData){
	$("#"+rowData.tdId).show();
 	$(".wirelessSetting").animate({scrollTop:0},0);
	$("#"+rowData.tdId + " li").html("IPSec Tunnel " + rowData.id);
 	$("#commonConfigViewOrModify input,select").prop("disabled",true);
}
function modifyChannel(rowData){
	var index = rowData.tdId.split('l')[1];
	$("select[name='TUNNEL_ENABLE_"+index +"']").val(rowData.zhangtai).attr('oldValue',rowData.zhangtai);
	$("input[name='TUNNEL_NAME_"+index +"']").val(rowData.mingcheng).attr('oldValue',rowData.mingcheng);
	$("input[name='TUNNEL_GATEWAY_"+index +"']").val(rowData.wangguan).attr('oldValue',rowData.wangguan);
	$("select[name='AUTHBY_"+index +"']").val(rowData.fangshi).attr('oldValue',rowData.fangshi);
	$("input[name='LEFT_IDENTIFIER_"+index+"']").attr('oldValue',$("input[name='LEFT_IDENTIFIER_"+index+"']").val());
	$("input[name='RIGHT_IDENTIFIER_"+index+"']").attr('oldValue',$("input[name='RIGHT_IDENTIFIER_"+index+"']").val());
	$("input[name='PRE_SHARED_KEY_"+index+"']").attr('oldValue',$("input[name='PRE_SHARED_KEY_"+index+"']").val()) ;
	$("input[name='LEFTSOURCEIP_"+index+"']").attr('oldValue',$("input[name='LEFTSOURCEIP_"+index+"']").val()) ;
	$("select[name='IKE_ENCRYPTION_"+index+"']").attr('oldValue',$("select[name='IKE_ENCRYPTION_"+index+"']").val())  ;
	$("select[name='ESP_ENCRYPTION_"+index+"']").attr('oldValue',$("select[name='ESP_ENCRYPTION_"+index+"']").val()) ;
	$("select[name='IKE_DH_GROUP_"+index+"']").attr('oldValue',$("select[name='IKE_DH_GROUP_"+index+"']").val()) ;
	$("select[name='ESP_DH_GROUP_"+index+"']").attr('oldValue',$("select[name='ESP_DH_GROUP_"+index+"']").val()) ;
	$("select[name='IKE_AUTHENTICATION_"+index+"']").attr('oldValue',$("select[name='IKE_AUTHENTICATION_"+index+"']").val()) ;
	$("select[name='ESP_AUTHENTICATION_"+index+"']").attr('oldValue',$("select[name='ESP_AUTHENTICATION_"+index+"']").val()) ;
	$("input[name='KEYLIFE_"+index+"']").attr('oldValue',$("input[name='KEYLIFE_"+index+"']").val());
	$("input[name='IKELIFETIME_"+index+"']").attr('oldValue',$("input[name='IKELIFETIME_"+index+"']").val()) ;
	$("input[name='REKEYMARGIN_"+index+"']").attr('oldValue',$("input[name='REKEYMARGIN_"+index+"']").val()) ;
	$("input[name='KEYINGTRIES_"+index+"']").attr('oldValue',$("input[name='KEYINGTRIES_"+index+"']").val()) ;
	$("select[name='DPDACTION_"+index+"']").attr('oldValue',$("select[name='DPDACTION_"+index+"']").val()) ;
	$("input[name='DPDDELAY_"+index+"']").attr('oldValue',$("input[name='DPDDELAY_"+index+"']").val()) ;
	$("input[name='ROOTCERTIFICATES_"+index+"']").attr('oldValue',$("input[name='ROOTCERTIFICATES_"+index+"']").val()) ;
	$("input[name='CERTIFICATES_"+index+"']").attr('oldValue',$("input[name='CERTIFICATES_"+index+"']").val()) ;
	$("input[name='PRIVATEKEYS_"+index+"']").attr('oldValue',$("input[name='PRIVATEKEYS_"+index+"']").val()) ;
	$("input[name='RIGHT_SUBNET_"+index+"']").attr('oldValue',$("input[name='RIGHT_SUBNET_"+index+"']").val()) ;
	$("#"+rowData.tdId).show();
	$("#"+rowData.tdId+" .linkbutton_trend").attr("isModify",'true').attr('tunnelId',rowData.id);
	$("#"+rowData.tdId + " li").html("IPSec Tunnel " + rowData.id);
 	$(".wirelessSetting").animate({scrollTop:0},0);
}
function deleteChannel(rowData){
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChu")%>", function (r) {
        if (r) {
        	dataList = dataList.filter(function(item){
        		return item.id != rowData.id
        	})
        	if(dataList.length>0){
        		dataList[0].id=1;
        	}
        	$("#ipsecTunnelData").datagrid({data : dataList});
			$("#tunnelAdd").removeClass('titleIcon_add_disabled').addClass('titleIcon_add');
			$("#ipsecTunnelData_err").hide();
        }
    }).addClass("seriousConfirm");
}
function saveTunnel1Modify(ele){
	var isPass = true;
	if($("#basic_TUNNEL_NAME_1").val()==""){
		isPass = false;
		$("#basic_TUNNEL_NAME_1").addClass('err_border');
		$("#basic_TUNNEL_NAME_1_err").show();
	}
	if($("#basic_TUNNEL_GATEWAY_1").val()==""){
		isPass = false;
		$("#basic_TUNNEL_GATEWAY_1").addClass('err_border');
		$("#basic_TUNNEL_GATEWAY_1_err").show();
	}
	if($("#basic_PRE_SHARED_KEY_1").val()==""){
		isPass = false;
		$("#basic_PRE_SHARED_KEY_1").addClass('err_border');
		$("#basic_PRE_SHARED_KEY_1_err").show();
	}
	if (!isPass || $("#channel1 .item.err_border").length > 0 ) {
	 	$(".wirelessSetting").animate({scrollTop:0},200);
		return;
	}
	
	dataList = dataList.filter(function(item){
		return item.tdId != 'channel1'
	})
	var isModify = $("#"+ele+" .linkbutton_trend").attr("isModify");
	var tunnelId = $("#"+ele+" .linkbutton_trend").attr("tunnelId");
	var dataListTunnel1= {};
	if(dataList.length == 0){
		dataListTunnel1.id = 1;
	}else{
		if(isModify != 'true'){
			dataListTunnel1.id = 2;
		}else{
			dataListTunnel1.id = tunnelId;
		}
	}
	dataListTunnel1.tdId = 'channel1';
	dataListTunnel1.mingcheng = $("#basic_TUNNEL_NAME_1").val();
	dataListTunnel1.wangguan = $("#basic_TUNNEL_GATEWAY_1").val();
	dataListTunnel1.fangshi = $("#basic_AUTHBY_1").val();
	dataListTunnel1.zhangtai = $("#basic_TUNNEL_ENABLE_1").val();
	if(tunnelId == '1'){
		dataList.unshift(dataListTunnel1);
	}else{
		dataList.push(dataListTunnel1);
	}
	$("#ipsecTunnelData").datagrid({data : dataList});
	$("#"+ele).hide();
	if(dataList.length == 2){
		$("#tunnelAdd").removeClass('titleIcon_add').addClass('titleIcon_add_disabled');
	}
	$("#ipsecTunnelData_err").hide();
}
function saveTunnel2Modify(ele){
	var isPass = true;
	if($("#basic_TUNNEL_NAME_2").val()==""){
		isPass = false;
		$("#basic_TUNNEL_NAME_2").addClass('err_border');
		$("#basic_TUNNEL_NAME_2_err").show();
	}
	if($("#basic_TUNNEL_GATEWAY_2").val()==""){
		isPass = false;
		$("#basic_TUNNEL_GATEWAY_2").addClass('err_border');
		$("#basic_TUNNEL_GATEWAY_2_err").show();
	}
	if($("#basic_PRE_SHARED_KEY_2").val()==""){
		isPass = false;
		$("#basic_PRE_SHARED_KEY_2").addClass('err_border');
		$("#basic_PRE_SHARED_KEY_2_err").show();
	}
	if (!isPass || $("#channel2 .item.err_border").length > 0 ) {
	 	$(".wirelessSetting").animate({scrollTop:0},200);
		return;
	}
	dataList = dataList.filter(function(item){
		return item.tdId != 'channel2'
	})
	var isModify = $("#"+ele+" .linkbutton_trend").attr("isModify");
	var tunnelId = $("#"+ele+" .linkbutton_trend").attr("tunnelId");
	var dataListTunnel2= {};
	if(dataList.length == 0){
		dataListTunnel2.id = 1;
	}else{
		if(isModify != 'true'){
			dataListTunnel2.id = 2;
		}else{
			dataListTunnel2.id = tunnelId;
		}
	}
	dataListTunnel2.tdId = 'channel2';
	dataListTunnel2.mingcheng = $("#basic_TUNNEL_NAME_2").val();
	dataListTunnel2.wangguan = $("#basic_TUNNEL_GATEWAY_2").val();
	dataListTunnel2.fangshi = $("#basic_AUTHBY_2").val();
	dataListTunnel2.zhangtai = $("#basic_TUNNEL_ENABLE_2").val();
	if(tunnelId == '1'){
		dataList.unshift(dataListTunnel2);
	}else{
		dataList.push(dataListTunnel2);
	}
	$("#ipsecTunnelData").datagrid({data : dataList});
	$("#"+ele).hide();
	if(dataList.length == 2){
		$("#tunnelAdd").removeClass('titleIcon_add').addClass('titleIcon_add_disabled');
	}
}
function cancelTunnelModify(ele){
	var index = ele.split('l')[1];
	$("#ele"+" input").removeClass('err_border');
	$("#basic_TUNNEL_NAME_1").removeClass('err_border');
	$(".errorTitle").hide();
	//创建ID文本框
	$("input[name='ID_"+index+"']").val(index) ;
	if($("select[name='TUNNEL_ENABLE_"+index +"']").attr('oldValue') != undefined){
		$("select[name='TUNNEL_ENABLE_"+index +"']").val($("select[name='TUNNEL_ENABLE_"+index +"']").attr('oldValue'));
	}
	if($("input[name='TUNNEL_NAME_"+index +"']").attr('oldValue') != undefined){
		$("input[name='TUNNEL_NAME_"+index +"']").val($("input[name='TUNNEL_NAME_"+index +"']").attr('oldValue')).removeClass('err_border');
	}
	if($("input[name='TUNNEL_GATEWAY_"+index +"']").attr('oldValue') != undefined){
		$("input[name='TUNNEL_GATEWAY_"+index +"']").val($("input[name='TUNNEL_GATEWAY_"+index +"']").attr('oldValue')).removeClass('err_border');
	}
	if($("input[name='LEFT_IDENTIFIER_"+index +"']").attr('oldValue') != undefined){
		$("input[name='LEFT_IDENTIFIER_"+index +"']").val($("input[name='LEFT_IDENTIFIER_"+index +"']").attr('oldValue')).removeClass('err_border');
	}
	if($("input[name='RIGHT_IDENTIFIER_"+index +"']").attr('oldValue') != undefined){
		$("input[name='RIGHT_IDENTIFIER_"+index +"']").val($("input[name='RIGHT_IDENTIFIER_"+index +"']").attr('oldValue')).removeClass('err_border');
	}
	if($("select[name='AUTHBY_"+index +"']").attr('oldValue') != undefined){
		$("select[name='AUTHBY_"+index +"']").val($("select[name='AUTHBY_"+index +"']").attr('oldValue'));
	}
	if($("input[name='PRE_SHARED_KEY_"+index +"']").attr('oldValue') != undefined){
		$("input[name='PRE_SHARED_KEY_"+index +"']").val($("input[name='PRE_SHARED_KEY_"+index +"']").attr('oldValue')).removeClass('err_border') ;
	}
	if($("input[name='LEFTSOURCEIP_"+index +"']").attr('oldValue') != undefined){
		$("input[name='LEFTSOURCEIP_"+index +"']").val($("input[name='LEFTSOURCEIP_"+index +"']").attr('oldValue')).removeClass('err_border') ;
	}
	if($("select[name='IKE_ENCRYPTION_"+index +"']").attr('oldValue') != undefined){
		$("select[name='IKE_ENCRYPTION_"+index +"']").val($("select[name='IKE_ENCRYPTION_"+index +"']").attr('oldValue'));
	}
	if($("select[name='ESP_ENCRYPTION_"+index +"']").attr('oldValue') != undefined){
		$("select[name='ESP_ENCRYPTION_"+index +"']").val($("select[name='ESP_ENCRYPTION_"+index +"']").attr('oldValue')) ;
	}
	if($("select[name='IKE_DH_GROUP_"+index +"']").attr('oldValue') != undefined){
		$("select[name='IKE_DH_GROUP_"+index +"']").val($("select[name='IKE_DH_GROUP_"+index +"']").attr('oldValue')) ;
	}
	if($("select[name='ESP_DH_GROUP_"+index +"']").attr('oldValue') != undefined){
		$("select[name='ESP_DH_GROUP_"+index +"']").val($("select[name='ESP_DH_GROUP_"+index +"']").attr('oldValue')) ;
	}
	if($("select[name='IKE_AUTHENTICATION_"+index +"']").attr('oldValue') != undefined){
		$("select[name='IKE_AUTHENTICATION_"+index +"']").val($("select[name='IKE_AUTHENTICATION_"+index +"']").attr('oldValue')) ;
	}
	if($("select[name='ESP_AUTHENTICATION_"+index +"']").attr('oldValue') != undefined){
		$("select[name='ESP_AUTHENTICATION_"+index +"']").val($("select[name='ESP_AUTHENTICATION_"+index +"']").attr('oldValue')) ;
	}
	if($("input[name='KEYLIFE_"+index +"']").attr('oldValue') != undefined){
		$("input[name='KEYLIFE_"+index +"']").val($("input[name='KEYLIFE_"+index +"']").attr('oldValue')).removeClass('err_border');
	}
	if($("input[name='IKELIFETIME_"+index +"']").attr('oldValue') != undefined){
		$("input[name='IKELIFETIME_"+index +"']").val($("input[name='IKELIFETIME_"+index +"']").attr('oldValue')).removeClass('err_border') ;
	}
	if($("input[name='REKEYMARGIN_"+index +"']").attr('oldValue') != undefined){
		$("input[name='REKEYMARGIN_"+index +"']").val($("input[name='REKEYMARGIN_"+index +"']").attr('oldValue')).removeClass('err_border') ;
	}
	if($("input[name='KEYINGTRIES_"+index +"']").attr('oldValue') != undefined){
		$("input[name='KEYINGTRIES_"+index +"']").val($("input[name='KEYINGTRIES_"+index +"']").attr('oldValue')).removeClass('err_border') ;
	}
	if($("select[name='DPDACTION_"+index +"']").attr('oldValue') != undefined){
		$("select[name='DPDACTION_"+index +"']").val($("select[name='DPDACTION_"+index +"']").attr('oldValue')) ;
	}
	if($("input[name='DPDDELAY_"+index +"']").attr('oldValue') != undefined){
		$("input[name='DPDDELAY_"+index +"']").val($("input[name='DPDDELAY_"+index +"']").attr('oldValue')).removeClass('err_border') ;
	}
	if($("input[name='ROOTCERTIFICATES_"+index +"']").attr('oldValue') != undefined){
		$("input[name='ROOTCERTIFICATES_"+index +"']").val($("input[name='ROOTCERTIFICATES_"+index +"']").attr('oldValue')).removeClass('err_border') ;
	}
	if($("input[name='CERTIFICATES_"+index +"']").attr('oldValue') != undefined){
		$("input[name='CERTIFICATES_"+index +"']").val($("input[name='CERTIFICATES_"+index +"']").attr('oldValue')).removeClass('err_border') ;
	}
	if($("input[name='RIGHT_SUBNET_"+index +"']").attr('oldValue') != undefined){
		$("input[name='PRIVATEKEYS_"+index +"']").val($("input[name='PRIVATEKEYS_"+index +"']").attr('oldValue')).removeClass('err_border') ;
	}
	if($("input[name='RIGHT_SUBNET_"+index +"']").attr('oldValue') != undefined){
		$("input[name='RIGHT_SUBNET_"+index +"']").val($("input[name='RIGHT_SUBNET_"+index +"']").attr('oldValue')).removeClass('err_border') ;
	}
	$("#"+ele).hide();
}
function addTunnel(){
	$("#ipsecTunnelData_err").hide();
	if(dataList.length==0){
		$("#channel1").show();
		$("#channel1 li").html("IPSec Tunnel 1");
		$("#channel1 input").val('');
		$("select[name='TUNNEL_ENABLE_1']").val('0');
		$("select[name='AUTHBY_1']").val('psk');
		$("select[name='IKE_ENCRYPTION_1']").val('');
		$("select[name='ESP_ENCRYPTION_1']").val('');
		$("select[name='IKE_DH_GROUP_1']").val('');
		$("select[name='ESP_DH_GROUP_1']").val('');
		$("select[name='IKE_AUTHENTICATION_1']").val('');
		$("select[name='ESP_AUTHENTICATION_1']").val('');
		$("select[name='DPDACTION_1']").val('');
		$("#channel1 .linkbutton_trend").attr("isModify",'false').attr('tunnelId','2');
	}else if(dataList.length==2){
		$("#ipsecTunnelData_err").show();
		return;
	}else{
		if(dataList[0].tdId == "channel1"){
			$("#channel2").show();
			$("#channel2 input").val('');
			$("#channel2 li").html("IPSec Tunnel 2");
			$("select[name='TUNNEL_ENABLE_2']").val('0');
			$("select[name='AUTHBY_2']").val('psk');
			$("select[name='IKE_ENCRYPTION_2']").val('');
			$("select[name='ESP_ENCRYPTION_2']").val('');
			$("select[name='IKE_DH_GROUP_2']").val('');
			$("select[name='ESP_DH_GROUP_2']").val('');
			$("select[name='IKE_AUTHENTICATION_2']").val('');
			$("select[name='ESP_AUTHENTICATION_2']").val('');
			$("select[name='DPDACTION_2']").val('');
			$("#channel2 .linkbutton_trend").attr("isModify",'false').attr('tunnelId','2');
		}else{
			$("#channel1").show();
			$("#channel1 input").val('');
			$("#channel1 li").html("IPSec Tunnel 2");
			$("select[name='TUNNEL_ENABLE_1']").val('0');
			$("select[name='AUTHBY_1']").val('psk');
			$("select[name='IKE_ENCRYPTION_1']").val('');
			$("select[name='ESP_ENCRYPTION_1']").val('');
			$("select[name='IKE_DH_GROUP_1']").val('');
			$("select[name='ESP_DH_GROUP_1']").val('');
			$("select[name='IKE_AUTHENTICATION_1']").val('');
			$("select[name='ESP_AUTHENTICATION_1']").val('');
			$("select[name='DPDACTION_1']").val('');
			$("#channel1 .linkbutton_trend").attr("isModify",'false').attr('tunnelId','2');
		}
	}
	$(".wirelessSetting").animate({scrollTop:0},0);
}
function tunnelSwitchStr(value, rowData, rowIndex){
	if(value == '1'){
		value = '<%=rb.getString("KaiQi")%>';
	}else{
		value = '<%=rb.getString("GuanBi")%>';
	}
	return value;
}
//变化时提示单站配置需要手工修改
function changeTipShowOrHide(e) {
	var ele = $(e["target"]);
	if(!ele.hasClass("err_border") && ele.val() != ele.attr('oldvalue')){
        $("#" + ele.attr("id") + "_tip").show();
	}else{
        $("#" + ele.attr("id") + "_tip").hide();
	}
}
</script>