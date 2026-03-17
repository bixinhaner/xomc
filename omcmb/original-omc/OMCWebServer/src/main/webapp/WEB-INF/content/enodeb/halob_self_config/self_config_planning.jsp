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
.ipsecChannel{
	position: absolute;
    background: #FFF;
    top: 0px;
    clear: both;
    display: none;
}
</style>
<div class="slidebarTitleDiv">
	<ul class="slidebarTitleContainer" style="margin-left:0;padding-left:0;">
		<li class="default" id="selfConfigTitle"></li>
	</ul>
	<div class="tableDiv titleIcon_close" onclick="closeSelfConfig();" style="position:absolute;right:25px;top:15px;"></div>
</div>
<div class="wirelessSetting" style="padding: 20px 0 0 20px;display:block">
	<div id="singleDeviceConfigDiv">
		<div class="itemDiv" style="">
			<span><%=rb.getString("TAC")%></span>
			<input type="text" name="LTE_TAC"  id="LTE_TAC" class="inputDivCss border border-box item" oldValue="" value=""
					onblur="validateMaxAndMinVal(event)" min_value="" max_value="" must="1" />
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="LTE_TAC_err"><%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 65535</div>	
		</div>
		<div class="itemDiv" style="">
			<span><%=rb.getString("JiZhanID")%></span>
			<input type="text" name="LTE_CELL_IDENTITY"  id="LTE_CELL_IDENTITY" class="inputDivCss border border-box item" oldValue="" value=""
					onblur="validateMaxAndMinVal(event)" min_value="" max_value="" must="1" />
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="LTE_CELL_IDENTITY_err"><%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> <%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> </div>	
		</div>
		<div class="itemDiv" style="">
			<span><%=rb.getString("PCI2")%></span>
			<input type="text" name="LTE_PHY_CELLID_LIST" id="LTE_PHY_CELLID_LIST" class="inputDivCss border border-box item" oldValue="" value=""
				onblur="validateByRegexAndRange(event)" min_value="" max_value="" must="1" 
				vali-regex="/^(\d+\.\.){0,1}(\d+)$/" />
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="LTE_PHY_CELLID_LIST_err"><%=rb.getString("ZhengXing")%>/<%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> <%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> </div>	
		</div>
		<div class="itemDiv" style="">
			<span><%=rb.getString("GenXuLieSuoYin")%></span>
			<input type="text" name="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST"  id="LTE_ROOT_SEQ_INDEX" class="inputDivCss border border-box item"  oldValue="" value="" 
				onblur="validateByRegexAndRange(event)" min_value="" max_value="" must="1" 
				vali-regex="/^(\d+\.\.){0,1}(\d+)$/" />
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="LTE_ROOT_SEQ_INDEX_err"><%=rb.getString("ZhengXing")%>/<%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> <%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> </div>	
		</div>
	</div>
	<div id="commonConfigDiv">
		<div class="itemDiv" style="">
			<span><%=rb.getString("ZhiChiPinDuan")%></span>
			<input type="text" name="LTE_BANDS_SUPPORTED" id="singleEnb_LTE_BANDS_SUPPORTED" class="inputDivCss border border-box item" 
		 		onblur="singleEnb_commonChangeTip(event)" title="stringList-[1:62]" min_value="1" max_value="62"/>					
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("DaiKuan")%></span>
			<select name="LTE_UL_BANDWIDTH" id="singleEnb_LTE_UL_BANDWIDTH" class="inputDivCss border border-box item">
				<option value="n25">5MHz</option>
				<option value="n50">10MHz</option>
				<option value="n75">15MHz</option>
				<option value="n100">20MHz</option>
			</select>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
			<input type="hidden" value="" id="singleEnb_OLD_LTE_UL_BANDWIDTH">
		</div>
		<div class="itemDiv pciisLock">
			<span><%=rb.getString("PinLv")%>(MHz)</span>
			<input type="text" name="LTE_UL_DL_EARFCN" id="singleEnb_LTE_UL_DL_EARFCN" class="inputDivCss border border-box item" 
					onblur="singleEnb_validateMaxAndMinVal1(event)" min_value="0" max_value="3800"
					title="int, min value: 0, max value: 3800"/>					
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("ZiZhenPeiBi")%></span>
			<select name="LTE_TDD_SUBFRAME_ASSIGNMENT" id="singleEnb_LTE_TDD_SUBFRAME_ASSIGNMENT" class="inputDivCss border border-box item">
				<option value="1">1(DL:UL = 2:2)</option>
				<option value="2">2(DL:UL = 3:1)</option>
			</select>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
			<input type="hidden" value="" id="singleEnb_OLD_LTE_TDD_SUBFRAME_ASSIGNMENT">
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("TeShuZiZhenPeiBi")%></span>
			<select name="LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS" id="singleEnb_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS" class="inputDivCss border border-box item">
				<option value="5">5</option>
				<option value="7">7</option>
			</select>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
			<input type="hidden" value="" id="singleEnb_OLD_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS">
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("PLMN")%></span>
			<input type="text" name="LTE_OAM_PLMNID" id="singleEnb_LTE_OAM_PLMNID" class="inputDivCss border border-box item"
					onblur="singleEnb_commonChangeTip(event)"  min_value="10000" max_value="999999"
					title="string, min value: 10000, max value: 999999"/>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>			
		</div>
		<div class="itemDiv" style="float:left;clear:both">
			<span><%=rb.getString("HeXinWang")%></span>
			<input type="text" name="singleEnb_LTE_SIGLINK_SERVER_LIST" id="singleEnb_LTE_SIGLINK_SERVER_LIST" class="inputDivCss border border-box item" 
				oldValue="" value="" onblur="singleEnb_validateIPAddress1(event)" />
			<div class='titleIcon titleIcon_add' style='cursor:pointer;display: inline-block;width:26px;height:26px;vertical-align:middle;' onclick="basic_addMMEIpInputText(this)"></div>
			<div class="errorTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("HaloBJiChuPeiZhiTitle")%></span>
			<input type="text" name="LTE_HALOB_ENABLE_STATE" id="singleEnb_LTE_HALOB_ENABLE_STATE" class="inputDivCss border border-box item" />					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>			
		</div>
		<!-- IPSEC设置 -->
		<div class="IPSECSetting" id="singleEnb_IPSECDIV">
			<div class="itemDiv">
				<span><%=rb.getString("IpsecKaiGuan")%></span>
				<select name="PSEC_ENABLE" id="singleEnb_PSEC_ENABLE" class="inputDivCss border border-box item">
					<option value="0"><%=rb.getString("GuanBi")%></option>
					<option value="1"><%=rb.getString("KaiQi")%></option>
				</select>
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<input type="hidden" value="" id="singleEnb_PSEC_ENABLE">
			</div>
			<div class="itemDiv">
				<span><%=rb.getString("IKEXieShangMuDiDuanKou")%></span>
				<select name="IPSEC_RIGHTIKEPORT" id="singleEnb_IPSEC_RIGHTIKEPORT" class="inputDivCss border border-box item">
					<option value="500">500</option>
					<option value="4500">4500</option>
				</select>
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<input type="hidden" value="" id="singleEnb_IPSEC_RIGHTIKEPORT">
			</div>
			<div class="itemDiv" style="">
				<span>Left Interface</span>
				<select name="RIGHT_LEFT_INTERFACE" id="singleEnb_RIGHT_LEFT_INTERFACE" class="inputDivCss border border-box item">
					<option value="none">none</option>
					<option value="WAN">WAN(eth2)</option>
					<option value="PPPOE">PPPOE(pppoe-wan)</option>
				</select>				
				<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
				<div class="errorTitle" id="singleEnb_LEFT_INTERFACE_err"> </div>	
			</div>
			<div style="margin-left:40px;clear:both;height:26px;line-height:26px;width:733px;">
				<span>IPSec Tunnel <%=rb.getString("LieBiao")%></span>
				<div class='titleIcon titleIcon_add' id='tunnelAdd' style='cursor:pointer;float:right;;width:26px;height:26px;vertical-align:middle;' onclick="addTunnel(this)"></div>
			</div>
			<div style="height:130px">
				<div style="height:103px;width:733px;border:1px solid #CCE1EF;float:left;margin-right:50px;margin-left:40px">
					<table id="singleEnb_ipsecTunnelData"></table>
				</div>
				<div class="errorTitle" id="singleEnb_ipsecTunnelData_err" style="clear:both;margin-left:40px;width:700px;height:26px;"></div>	
			</div>
			<!-- 索引1的内容 -->
			<div id="channel_single_1" class="ipsecChannel">
				<div class="omcPageTitleDiv" style="margin-bottom: 20px;">
					<ul class="omcPageTitleContainer">
						<li class="default">IPSec Tunnel1</li>
					</ul>
				</div>
				<div class="itemDiv" style="display:none">
					<span><%=rb.getString("SuoYin") %></span>
					<input type="text" name="ID_1" id="ID_1" class="inputDivCss border border-box item" disabled value = '1'/>					
					<div class="errorTitle"> </div>	
				</div> 
				<div class="itemDiv">
					<span><%=rb.getString("TunnelKaiGuan")%></span>
					<select name="TUNNEL_ENABLE_1" id="singleEnb_TUNNEL_ENABLE_1" class="inputDivCss border border-box item">
						<option value="0"><%=rb.getString("GuanBi")%></option>
						<option value="1"><%=rb.getString("KaiQi")%></option>
					</select>
					<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv" style="">
					<span><%=rb.getString("TunnelMingCheng")%></span>
					<input type="text" name="TUNNEL_NAME_1" id="singleEnb_TUNNEL_NAME_1" class="inputDivCss border border-box item" 
					min_length="1" max_length="14" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="/^\w+$/" />					
					<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
					<div class="errorTitle" id="singleEnb_TUNNEL_NAME_1_err" ><%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChangDu")%>14</div>	
				</div>
				<div class="itemDiv" style="">
					<span><%=rb.getString("TunnelWangGuan")%></span>
					<input type="text" name="TUNNEL_GATEWAY_1" id="singleEnb_TUNNEL_GATEWAY_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);"  onfocus="removeErrTip(event);"
					js_regex="no_zh" />					
					<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
					<div class="errorTitle" id="singleEnb_TUNNEL_GATEWAY_1_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>1-256</div>	
				</div>
				<div class="itemDiv">
					<span><%=rb.getString("RenZhengFangShi") %></span>
					<select name="AUTHBY_1" id="singleEnb_AUTHBY_1" class="inputDivCss border border-box item">
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
					<input type="password" name="PRE_SHARED_KEY_1" id="singleEnb_PRE_SHARED_KEY_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex = "no_zh" />
					<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
					<div class="errorTitle"  id="singleEnb_PRE_SHARED_KEY_1_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>1-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>Left Identifier</span>
					<input type="text" name="LEFT_IDENTIFIER_1" id="singleEnb_LEFT_IDENTIFIER_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="no_zh"/>					
					<div class="errorTitle" id="singleEnb_LEFT_IDENTIFIER_1_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>Right Identifier</span>
					<input type="text" name="RIGHT_IDENTIFIER_1" id="singleEnb_RIGHT_IDENTIFIER_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="no_zh"/>				
					<div class="errorTitle" id="singleEnb_RIGHT_IDENTIFIER_1_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>leftsourceip</span>
					<input type="text" name="LEFTSOURCEIP_1" id="singleEnb_LEFTSOURCEIP_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="/^(?:(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])|%config)$/"/>					
					<div class="errorTitle" id="singleEnb_LEFTSOURCEIP_1_err" ><%=rb.getString("IPDiZhiHuoConfig")%></div>	
				</div>
				<div class="itemDiv">
					<span>Ike Encryption</span>
					<select name="IKE_ENCRYPTION_1" id="singleEnb_IKE_ENCRYPTION_1" class="inputDivCss border border-box item">
						<option value="null">null</option>
						<option value="aes128">aes128</option>
						<option value="aes256">aes256</option>
						<option value="3des">3des</option>
					</select>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv">
					<span>ESP Encryption</span>
					<select name="ESP_ENCRYPTION_1" id="singleEnb_ESP_ENCRYPTION_1" class="inputDivCss border border-box item">
						<option value="null">null</option>
						<option value="aes128">aes128</option>
						<option value="aes256">aes256</option>
						<option value="3des">3des</option>
					</select>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv">
					<span>Ike Dh Group</span>
					<select name="IKE_DH_GROUP_1" id="singleEnb_IKE_DH_GROUP_1" class="inputDivCss border border-box item">
						<option value="null">null</option>
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
					<select name="ESP_DH_GROUP_1" id="singleEnb_ESP_DH_GROUP_1" class="inputDivCss border border-box item">
						<option value="null">null</option>
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
					<select name="IKE_AUTHENTICATION_1" id="singleEnb_IKE_AUTHENTICATION_1" class="inputDivCss border border-box item">
						<option value="null">null</option>
						<option value="sha1">sha1</option>
						<option value="sha512">sha512</option>
					</select>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv">
					<span>Esp Authentication</span>
					<select name="ESP_AUTHENTICATION_1" id="singleEnb_ESP_AUTHENTICATION_1" class="inputDivCss border border-box item">
						<option value="null">null</option>
						<option value="sha1">sha1</option>
					</select>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv" style="">
					<span>Keylife</span>
					<input type="text" name="KEYLIFE_1" id="singleEnb_KEYLIFE_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="/^\d+[s|m|d]{1}$/"/>					
					<div class="errorTitle" id="singleEnb_KEYLIFE_1_err" ><%=rb.getString("ShuZiJiaSMD")%></div>	
				</div>
				<div class="itemDiv" style="">
					<span>Ikelifetime</span>
					<input type="text" name="IKELIFETIME_1" id="singleEnb_IKELIFETIME_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="/^\d+[s|m|d]{1}$/"/>					
					<div class="errorTitle" id="singleEnb_IKELIFETIME_1_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
				</div>
				<div class="itemDiv" style="">
					<span>Rekeymargin</span>
					<input type="text" name="REKEYMARGIN_1" id="singleEnb_REKEYMARGIN_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="/^\d+[s|m|d]{1}$/"/>					
					<div class="errorTitle" id="singleEnb_REKEYMARGIN_1_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
				</div>
				<div class="itemDiv" style="">
					<span>Keyingtries</span>
					<input type="text" name="KEYINGTRIES_1" id="singleEnb_KEYINGTRIES_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="/^(?:\d+|%forever)$/"/>					
					<div class="errorTitle" id="singleEnb_KEYINGTRIES_1_err"><%=rb.getString("ShuZiHuoforever")%></div>	
				</div>
				<div class="itemDiv">
					<span>dpdaction</span>
					<select name="DPDACTION_1" id="singleEnb_DPDACTION_1" class="inputDivCss border border-box item">
						<option value="null">null</option>
						<option value="none">none</option>
						<option value="clear">clear</option>
						<option value="hold">hold</option>
						<option value="restart">restart</option>
					</select>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv" style="">
					<span>Dpddelay</span>
					<input type="text" name="DPDDELAY_1" id="singleEnb_DPDDELAY_1" class="inputDivCss border border-box item"
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="/^\d+[s|m|d]{1}$/"/>				
					<div class="errorTitle" id="singleEnb_DPDDELAY_1_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
				</div>
				<div class="itemDiv" style="">
					<span>Rootcertificates</span>
					<input type="text" name="ROOTCERTIFICATES_1" id="singleEnb_ROOTCERTIFICATES_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="no_zh"/>					
					<div class="errorTitle" id="singleEnb_ROOTCERTIFICATES_1_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>Certificates</span>
					<input type="text" name="CERTIFICATES_1" id="singleEnb_CERTIFICATES_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="no_zh"/>					
					<div class="errorTitle" id="singleEnb_CERTIFICATES_1_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>Privatekeys</span>
					<input type="text" name="PRIVATEKEYS_1" id="singleEnb_PRIVATEKEYS_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="no_zh"/>					
					<div class="errorTitle" id="singleEnb_PRIVATEKEYS_1_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>RightSubnet</span>
					<input type="text" name="RIGHT_SUBNET_1" id="singleEnb_RIGHT_SUBNET_1" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="no_zh"/>				
					<div class="errorTitle" id="singleEnb_RIGHT_SUBNET_1_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<a href="#" class="linkbutton" style="margin-left:40px;" onclick="cancelTunnelModify('channel_single_1')"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
			<div id="channel_single_2" class="ipsecChannel">
				<div class="omcPageTitleDiv" style="margin-bottom: 20px;">
					<ul class="omcPageTitleContainer">
						<li class="default">IPSec Tunnel1</li>
					</ul>
				</div>
				<div class="itemDiv" style="display:none">
					<span><%=rb.getString("SuoYin") %></span>
					<input type="text" name="ID_2" id="ID_2" class="inputDivCss border border-box item" disabled value = '2'/>					
					<div class="errorTitle"> </div>	
				</div> 
				<div class="itemDiv">
					<span><%=rb.getString("TunnelKaiGuan")%></span>
					<select name="TUNNEL_ENABLE_2" id="singleEnb_TUNNEL_ENABLE_2" class="inputDivCss border border-box item">
						<option value="0"><%=rb.getString("GuanBi")%></option>
						<option value="1"><%=rb.getString("KaiQi")%></option>
					</select>
					<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv" style="">
					<span><%=rb.getString("TunnelMingCheng")%></span>
					<input type="text" name="TUNNEL_NAME_2" id="singleEnb_TUNNEL_NAME_2" class="inputDivCss border border-box item" 
					min_length="1" max_length="14" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="/^\w+$/" />					
					<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
					<div class="errorTitle" id="singleEnb_TUNNEL_NAME_2_err" ><%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChangDu")%>14</div>	
				</div>
				<div class="itemDiv" style="">
					<span><%=rb.getString("TunnelWangGuan")%></span>
					<input type="text" name="TUNNEL_GATEWAY_2" id="singleEnb_TUNNEL_GATEWAY_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);"  onfocus="removeErrTip(event);"
					js_regex="no_zh" />					
					<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
					<div class="errorTitle" id="singleEnb_TUNNEL_GATEWAY_2_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>1-256</div>	
				</div>
				<div class="itemDiv">
					<span><%=rb.getString("RenZhengFangShi") %></span>
					<select name="AUTHBY_2" id="singleEnb_AUTHBY_2" class="inputDivCss border border-box item">
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
					<input type="password" name="PRE_SHARED_KEY_2" id="singleEnb_PRE_SHARED_KEY_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" must="1" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex = "no_zh" />
					<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
					<div class="errorTitle"  id="singleEnb_PRE_SHARED_KEY_2_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>1-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>Left Identifier</span>
					<input type="text" name="LEFT_IDENTIFIER_2" id="singleEnb_LEFT_IDENTIFIER_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="no_zh"/>					
					<div class="errorTitle" id="singleEnb_LEFT_IDENTIFIER_2_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>Right Identifier</span>
					<input type="text" name="RIGHT_IDENTIFIER_2" id="singleEnb_RIGHT_IDENTIFIER_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="no_zh"/>				
					<div class="errorTitle" id="singleEnb_RIGHT_IDENTIFIER_2_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>leftsourceip</span>
					<input type="text" name="LEFTSOURCEIP_2" id="singleEnb_LEFTSOURCEIP_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="/^(?:(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])|%config)$/"/>					
					<div class="errorTitle" id="singleEnb_LEFTSOURCEIP_2_err" ><%=rb.getString("IPDiZhiHuoConfig")%></div>	
				</div>
				<div class="itemDiv">
					<span>Ike Encryption</span>
					<select name="IKE_ENCRYPTION_2" id="singleEnb_IKE_ENCRYPTION_2" class="inputDivCss border border-box item">
						<option value="null">null</option>
						<option value="aes128">aes128</option>
						<option value="aes256">aes256</option>
						<option value="3des">3des</option>
					</select>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv">
					<span>ESP Encryption</span>
					<select name="ESP_ENCRYPTION_2" id="singleEnb_ESP_ENCRYPTION_2" class="inputDivCss border border-box item">
						<option value="null">null</option>
						<option value="aes128">aes128</option>
						<option value="aes256">aes256</option>
						<option value="3des">3des</option>
					</select>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv">
					<span>Ike Dh Group</span>
					<select name="IKE_DH_GROUP_2" id="singleEnb_IKE_DH_GROUP_2" class="inputDivCss border border-box item">
						<option value="null">null</option>
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
					<select name="ESP_DH_GROUP_2" id="singleEnb_ESP_DH_GROUP_2" class="inputDivCss border border-box item">
						<option value="null">null</option>
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
					<select name="IKE_AUTHENTICATION_2" id="singleEnb_IKE_AUTHENTICATION_2" class="inputDivCss border border-box item">
						<option value="null">null</option>
						<option value="sha1">sha1</option>
						<option value="sha512">sha512</option>
					</select>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv">
					<span>Esp Authentication</span>
					<select name="ESP_AUTHENTICATION_2" id="singleEnb_ESP_AUTHENTICATION_2" class="inputDivCss border border-box item">
						<option value="null">null</option>
						<option value="sha1">sha1</option>
					</select>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv" style="">
					<span>Keylife</span>
					<input type="text" name="KEYLIFE_2" id="singleEnb_KEYLIFE_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="/^\d+[s|m|d]{1}$/"/>					
					<div class="errorTitle" id="singleEnb_KEYLIFE_2_err" ><%=rb.getString("ShuZiJiaSMD")%></div>	
				</div>
				<div class="itemDiv" style="">
					<span>Ikelifetime</span>
					<input type="text" name="IKELIFETIME_2" id="singleEnb_IKELIFETIME_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="/^\d+[s|m|d]{1}$/"/>					
					<div class="errorTitle" id="singleEnb_IKELIFETIME_2_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
				</div>
				<div class="itemDiv" style="">
					<span>Rekeymargin</span>
					<input type="text" name="REKEYMARGIN_2" id="singleEnb_REKEYMARGIN_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);"
					js_regex="/^\d+[s|m|d]{1}$/"/>					
					<div class="errorTitle" id="singleEnb_REKEYMARGIN_2_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
				</div>
				<div class="itemDiv" style="">
					<span>Keyingtries</span>
					<input type="text" name="KEYINGTRIES_2" id="singleEnb_KEYINGTRIES_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="/^(?:\d+|%forever)$/"/>					
					<div class="errorTitle" id="singleEnb_KEYINGTRIES_2_err"><%=rb.getString("ShuZiHuoforever")%></div>	
				</div>
				<div class="itemDiv">
					<span>dpdaction</span>
					<select name="DPDACTION_2" id="singleEnb_DPDACTION_2" class="inputDivCss border border-box item">
						<option value="null">null</option>
						<option value="none">none</option>
						<option value="clear">clear</option>
						<option value="hold">hold</option>
						<option value="restart">restart</option>
					</select>
					<div class="errorTitle"> </div>
				</div>
				<div class="itemDiv" style="">
					<span>Dpddelay</span>
					<input type="text" name="DPDDELAY_2" id="singleEnb_DPDDELAY_2" class="inputDivCss border border-box item"
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="/^\d+[s|m|d]{1}$/"/>				
					<div class="errorTitle" id="singleEnb_DPDDELAY_2_err"><%=rb.getString("ShuZiJiaSMD")%></div>	
				</div>
				<div class="itemDiv" style="">
					<span>Rootcertificates</span>
					<input type="text" name="ROOTCERTIFICATES_2" id="singleEnb_ROOTCERTIFICATES_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="no_zh"/>					
					<div class="errorTitle" id="singleEnb_ROOTCERTIFICATES_2_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>Certificates</span>
					<input type="text" name="CERTIFICATES_2" id="singleEnb_CERTIFICATES_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="no_zh"/>					
					<div class="errorTitle" id="singleEnb_CERTIFICATES_2_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>Privatekeys</span>
					<input type="text" name="PRIVATEKEYS_2" id="singleEnb_PRIVATEKEYS_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="no_zh"/>					
					<div class="errorTitle" id="singleEnb_PRIVATEKEYS_2_err" ><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<div class="itemDiv" style="">
					<span>RightSubnet</span>
					<input type="text" name="RIGHT_SUBNET_2" id="singleEnb_RIGHT_SUBNET_2" class="inputDivCss border border-box item" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);specialChar(event);" onfocus="removeErrTip(event);" 
					js_regex="no_zh"/>				
					<div class="errorTitle" id="singleEnb_RIGHT_SUBNET_2_err"><%=rb.getString("FeiZhongWenZiFu")%><%=rb.getString("ZiFuChangDu")%>0-256</div>	
				</div>
				<a href="#" class="linkbutton" style="margin-left:40px;" onclick="cancelTunnelModify('channel_single_2')"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
			<div class="linkbuttonGroup" id="buttonGroupView_single" style="margin: 40px 0 20px 40px;float:left;">
			    <a id="halobConfigSub" href="#" class="linkbutton linkbutton_trend" onclick="commonConfigSubmit(this)" ><span><%=rb.getString("QueDing")%></span></a>
			    <a href="#" class="linkbutton linkbutton_nowanna" onclick="closeSelfConfig()"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
			<a class="selfConfigSucTip" style="display:none;margin-bottom: 20px;margin-top:40px;"><%=rb.getString("DanZhanPeiZhiXiuGaiWanCheng")%></a>
		</div>
	</div>	
</div>
<script type="text/javascript">
var channelNumStr = "";
var dataList = [];
var mmeNum = '';
$(function() {
	var cellIpsecInfos = "";
	if(singleDeviceIsView == "true"){
		$("#buttonGroupView_single").hide();
	}
	$("#singleEnb_ipsecTunnelData").datagrid({
		border:false,
    	fit:true,
        singleSelect:true,
        rownumbers:false,
        fitColumns: false,
        striped:true,
        columns: [[
                   {field: 'tdId',hidden:true},
                   {field: 'id',width: 100, title: 'Tunnel ID'},
                   {field: 'zhangtai',width: 100, title: '<%=rb.getString("TunnelKaiGuan")%>',formatter:selfTunnelSwitchStr},
                   {field: 'mingcheng',width: 165,title: '<%=rb.getString("TunnelMingCheng")%>'},
                   {field: 'wangguan',width: 165, title: '<%=rb.getString("TunnelWangGuan")%>'},
                   {field: 'fangshi',width: 103, title: '<%=rb.getString("RenZhengFangShi")%>'},
                   {field: 'operate',fixed:true,width: 100,title:'<%=rb.getString("CaoZuo")%>',formatter: singleEnb_viewTunnel},
                   
        ]],
        data : dataList
	})
	closeLoading();
	var serial_number = $("#singleDeviceConfig").datagrid("getSelected").serial_number;
	var params={};
	params.serial_number = serial_number;
	$.post("${ctx}/cell/halobSelfConfig/getSingleCellParamContent.action", params, function(data){
		if(data){
			//tac
			if(data.tac_max_value == data.tac_min_value){
				var LTE_TAC_title = "<%=rb.getString("ZhengXing")%><%=rb.getString("MaoHao")%> " + data.tac_min_value;
			}else{
				var LTE_TAC_title = "<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> "+data.tac_min_value+"<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> " + data.tac_max_value;
			}
			$("#LTE_TAC").attr('min_value',data.tac_min_value).attr('max_value',data.tac_max_value).attr('title',LTE_TAC_title);
			$("#LTE_TAC_err").html(LTE_TAC_title);
			//cell id 
			if(data.cell_id_max_value == data.cell_id_min_value){
				var LTE_CELL_IDENTITY_title = "<%=rb.getString("ZhengXing")%><%=rb.getString("MaoHao")%> " + data.cell_id_max_value;
			}else{
				var LTE_CELL_IDENTITY_title = "<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> "+data.cell_id_min_value+"<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> " + data.cell_id_max_value;
			}
			$("#LTE_CELL_IDENTITY").attr('min_value',data.cell_id_min_value).attr('max_value',data.cell_id_max_value).attr('title',LTE_CELL_IDENTITY_title);
			$("#LTE_CELL_IDENTITY_err").html(LTE_CELL_IDENTITY_title);
			//pci
			if(data.pci_max_value == data.pci_min_value){
				var LTE_PHY_CELLID_LIST_title = "<%=rb.getString("ZhengXing")%><%=rb.getString("MaoHao")%> " + data.pci_max_value;
				var LTE_PHY_CELLID_LIST_tip = "<%=rb.getString("ZhengXing")%><%=rb.getString("MaoHao")%> " + data.pci_max_value;
			}else{
				var LTE_PHY_CELLID_LIST_title = "<%=rb.getString("LiRu")%>: '23' or '12..34'<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%>/<%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> "+ data.pci_min_value +"<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> " + data.pci_max_value;
				var LTE_PHY_CELLID_LIST_tip = "<%=rb.getString("ZhengXing")%>/<%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> "+ data.pci_min_value +"<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> " + data.pci_max_value;
			}
			$("#LTE_PHY_CELLID_LIST").attr('min_value',data.pci_min_value).attr('max_value',data.pci_max_value).attr('title',LTE_PHY_CELLID_LIST_title);
			$("#LTE_PHY_CELLID_LIST_err").html(LTE_PHY_CELLID_LIST_tip);
			//RootSequenceIndex
			if(data.root_seq_index_max_value == data.root_seq_index_min_value){
				var LTE_ROOT_SEQ_INDEX_title = "<%=rb.getString("ZhengXing")%><%=rb.getString("MaoHao")%> " + data.root_seq_index_max_value;
				var LTE_ROOT_SEQ_INDEX_tip = "<%=rb.getString("ZhengXing")%><%=rb.getString("MaoHao")%> " + data.root_seq_index_max_value;
			}else{
				var LTE_ROOT_SEQ_INDEX_title = "<%=rb.getString("LiRu")%>: '23' or '12..34'<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%>/<%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> "+ data.root_seq_index_min_value +"<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> " + data.root_seq_index_max_value;
				var LTE_ROOT_SEQ_INDEX_tip = "<%=rb.getString("ZhengXing")%>/<%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> "+ data.root_seq_index_min_value +"<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> " + data.root_seq_index_max_value;
			}
			$("#LTE_ROOT_SEQ_INDEX").attr('min_value',data.root_seq_index_min_value).attr('max_value',data.root_seq_index_max_value).attr('title',LTE_ROOT_SEQ_INDEX_title);
			$("#LTE_ROOT_SEQ_INDEX_err").html(LTE_ROOT_SEQ_INDEX_tip);
			
			$("input[name='LTE_TAC']").val(data.tac);
			$("input[name='LTE_CELL_IDENTITY']").val(data.cell_identity);
			$("input[name='LTE_PHY_CELLID_LIST']").val(data.phycellid);
			$("input[name='LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST']").val(data.root_sequence_index);
			$("input[name='LTE_BANDS_SUPPORTED']").val(data.bands_support);
			$("select[name='LTE_UL_BANDWIDTH']").val(data.band_width);
			$("input[name='LTE_UL_DL_EARFCN']").val(data.frequency).attr('oldValue',data.frequency);
			$("select[name='LTE_TDD_SUBFRAME_ASSIGNMENT']").val(data.subframe_assignment);
			$("select[name='LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS']").val(data.special_subframe_patterns);
			$("input[name='LTE_OAM_PLMNID']").val(data.plmn_id);
			var mmeArr = data.mme_ip;
			if(data.mme_ip){
				mmeArr = data.mme_ip.split(",");
			}
			//当mme地址为n且n>1时，需要重新生成n-1个输入框
			if (mmeArr && mmeArr.length > 1) {
				//初始化第一个输入框的值和oldValue值
				var firstMMEDiv = $("input[name='singleEnb_LTE_SIGLINK_SERVER_LIST']:first");
				firstMMEDiv.val(mmeArr[0]);
				firstMMEDiv.attr("oldValue", mmeArr[0]);
				
				var newMMEInputCount = mmeArr.length;
				//生成n-1个输入框
				for(var count = 1; count < newMMEInputCount; count++) {
					//添加新的输入框
					var $mmeSpan = $("<span style='display:inline-block;visibility:hidden'><%=rb.getString("HeXinWang")%></span>");
					var $mmeDiv = $("<div class='itemDiv'></div>");
					var $mmeInput = $("<input type='text' name='singleEnb_LTE_SIGLINK_SERVER_LIST' id='singleEnb_SIGLINK_SERVER_LIST_"+count+"' class='inputDivCss border border-box item' "
						 + "onblur='validateIPAddress(event)' oldValue='" + mmeArr[count] + "' value='" + mmeArr[count] + "' style='margin-left:0px'></input>");
					var $mmeButtn = $("<div class='titleIcon titleIcon_sub' style='cursor:pointer;display: inline-block;width:26px;height:26px;vertical-align:middle;' onclick='singleEnb_removeMMEIpInputText(this)'></div>" 
							+"<div class='errorTitle' id='singleEnb_SIGLINK_SERVER_LIST_" + count + "_err'><%=rb.getString("IPDiZhi")%></div>");
					
					$mmeDiv.append($mmeSpan);
					$mmeDiv.append($mmeInput);
					$mmeDiv.append($mmeButtn);
					$mmeDiv.addClass("clearBoth");
					
					//将新增的输入框插入到plmn参数之前
					var plmnidDiv = $("#singleEnb_IPSECDIV");
					$mmeDiv.insertBefore(plmnidDiv);
				}
			} else {
				var firstMMEDiv = $("input[name='singleEnb_LTE_SIGLINK_SERVER_LIST']:first");
				firstMMEDiv.val(mmeArr);
				firstMMEDiv.attr("oldValue", mmeArr);
			}
			//halob开关
			if(data.halob_enable == "0"){
				$("#singleEnb_LTE_HALOB_ENABLE_STATE").val("<%=rb.getString("GuanBi")%>")
			}else{
				$("#singleEnb_LTE_HALOB_ENABLE_STATE").val("<%=rb.getString("KaiQi")%>")
			}
			$("select[name='PSEC_ENABLE']").val(data.ipsec_enable);
			//ipsec开关
			$("select[name='IPSEC_RIGHTIKEPORT']").val(data.ipsec_rightikeport);
			cellIpsecInfos = data.ipsecList;
			if(cellIpsecInfos.length== 1){
				$("#channel_single_2").hide();
			}
			if(cellIpsecInfos.length > 0 ){
				for(var i=0;i<cellIpsecInfos.length;i++){
					if(i==0){
						channelNumStr = "1";
						var dataListTunnel1={};
						var ID = cellIpsecInfos[i].ipsec_index;
						dataListTunnel1.tdId = 'channel_single_'+1;
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
						dataListTunnel2.tdId = 'channel_single_'+2;
						dataListTunnel2.id = 2;
						dataListTunnel2.mingcheng = cellIpsecInfos[i].tunnel_name;
						dataListTunnel2.wangguan = cellIpsecInfos[i].tunnel_gateway;
						dataListTunnel2.fangshi = cellIpsecInfos[i].authby;
						dataListTunnel2.zhangtai = cellIpsecInfos[i].tunnel_enable;
						dataList.push(dataListTunnel2);
					}
					//创建ID文本框
					$("input[name='ID_"+(i+1)+"']").val(ID) ;
					$("select[name='TUNNEL_ENABLE_"+(i+1)+"']").val(cellIpsecInfos[i].tunnel_enable);
					$("input[name='TUNNEL_NAME_"+(i+1)+"']").val(cellIpsecInfos[i].tunnel_name);
					$("input[name='TUNNEL_GATEWAY_"+(i+1)+"']").val(cellIpsecInfos[i].tunnel_gateway);
					$("input[name='LEFT_IDENTIFIER_"+(i+1)+"']").val(cellIpsecInfos[i].left_identifier);
					$("input[name='RIGHT_IDENTIFIER_"+(i+1)+"']").val(cellIpsecInfos[i].right_identifier);
					$("select[name='AUTHBY_"+(i+1)+"']").val(cellIpsecInfos[i].authby);
					$("input[name='PRE_SHARED_KEY_"+(i+1)+"']").val(cellIpsecInfos[i].pre_shared_key) ;
					$("input[name='LEFTSOURCEIP_"+(i+1)+"']").val(cellIpsecInfos[i].leftsourceip) ;
					$("select[name='IKE_ENCRYPTION_"+(i+1)+"']").val(cellIpsecInfos[i].ike_encryption) ;
					$("select[name='ESP_ENCRYPTION_"+(i+1)+"']").val(cellIpsecInfos[i].esp_encryption) ;
					$("select[name='IKE_DH_GROUP_"+(i+1)+"']").val(cellIpsecInfos[i].ike_dh_group) ;
					$("select[name='ESP_DH_GROUP_"+(i+1)+"']").val(cellIpsecInfos[i].esp_dh_group) ;
					$("select[name='IKE_AUTHENTICATION_"+(i+1)+"']").val(cellIpsecInfos[i].ike_authentication) ;
					$("select[name='ESP_AUTHENTICATION_"+(i+1)+"']").val(cellIpsecInfos[i].esp_authentication) ;
					$("input[name='KEYLIFE_"+(i+1)+"']").val(cellIpsecInfos[i].keylife) ;
					$("input[name='IKELIFETIME_"+(i+1)+"']").val(cellIpsecInfos[i].ikelifetime) ;
					$("input[name='REKEYMARGIN_"+(i+1)+"']").val(cellIpsecInfos[i].rekeymargin) ;
					$("input[name='KEYINGTRIES_"+(i+1)+"']").val(cellIpsecInfos[i].keyingtries) ;
					$("select[name='DPDACTION_"+(i+1)+"']").val(cellIpsecInfos[i].dpdaction) ;
					$("input[name='DPDDELAY_"+(i+1)+"']").val(cellIpsecInfos[i].dpddelay) ;
					$("input[name='ROOTCERTIFICATES_"+(i+1)+"']").val(cellIpsecInfos[i].rootcertificates) ;
					$("input[name='CERTIFICATES_"+(i+1)+"']").val(cellIpsecInfos[i].certificates) ;
					$("input[name='PRIVATEKEYS_"+(i+1)+"']").val(cellIpsecInfos[i].privatekeys) ;
					$("input[name='RIGHT_SUBNET_"+(i+1)+"']").val(cellIpsecInfos[i].right_subnet) ;
				}
				$("#singleEnb_ipsecTunnelData").datagrid({data : dataList});
			}
		}
		if(singleDeviceIsView == "true"){
			$("#buttonGroupView_single").hide();
			$("#selfConfigViewOrModify input,select").prop("disabled",true);
			$(".titleIcon_add,.titleIcon_sub").hide();
		}else{
			$("#commonConfigDiv input,select").prop("disabled",true);
			$("#commonConfigDiv .titleIcon_add,.titleIcon_sub").hide();
		}
	},"json");
})
var XiuGai = '<%=rb.getString("XiuGai")%>';
//操作列
function singleEnb_viewTunnel(value, rowData, rowIndex){
    var gridData = JSON.stringify(rowData);
	value = "<div class='operationDiv operation_view' title='"+ChaKan+"' onclick='viewSelfConfigChannel(" + gridData + ")'></div>";
	return value;
}
function viewSelfConfigChannel(rowData){
	$("#"+rowData.tdId).show();
 	$(".wirelessSetting").animate({scrollTop:0},0);
	$("#"+rowData.tdId + " li").html("IPSec Tunnel " + rowData.id);
 	$("#commonConfigViewOrModify input,select").prop("disabled",true);
}
function singleEnb_commonChangeTip(e){
    validateMaxAndMinVal(e);
	if ($("#singleEnb_LTE_BANDS_SUPPORTED").val().trim()=="") {
		$("#singleEnb_LTE_BANDS_SUPPORTED").addClass("err_border");
	}
	if ($("#singleEnb_LTE_OAM_PLMNID").val().trim()=="") {
		$("#singleEnb_LTE_OAM_PLMNID").addClass("err_border");
	}
}
function singleEnb_createIpsecInputPasswordDiv(name,filedName,value,disabled){
	var tip = singleEnb_createTitleForIpSec(filedName);
	if(disabled == true){
		var div = "<div class='itemDiv'><span>"+name+"</span><input type='password' name='"+filedName+"' class='inputDivCss border border-box item' oldValue='"
		+value+"' value='"+value+"' disabled='disabled'/></div>";
	}else{
		var div = "<div class='itemDiv'><span>"+name+"</span><input type='password' id='singleEnb_" + filedName + "' onblur='singleEnb_commonChangeTip(event)' name='"+filedName+"' class='inputDivCss border border-box item'  oldValue='"
	   +value+"' value='"+value+"'/></div>";
	}
	return $(div);
}
function singleEnb_createTitleForIpSec(fieldName){
	return fieldName;
}
//添加一个MME地址输入框
function singleEnb_addMMEIpInputText(e) {
	//添加新的输入框
	var oldValue =$("input[id='LTE_SIGLINK_SERVER_LIST']").first().attr("oldValue");
	var $mmeSpan = $("<span style='display:inline-block;visibility:hidden'><%=rb.getString("HeXinWang")%></span>");
	var $mmeDiv = $("<div class='itemDiv'></div>");
	var $mmeInput = $("<input type='text' name='singleEnb_LTE_SIGLINK_SERVER_LIST' class='inputDivCss border border-box item' "
		 + "id='"+Math.random()+"' onblur='singleEnb_validateIPAddress11(event)'  oldValue='"+oldValue+"' style='margin-left:0px'></input>");
	var $mmeButtn = $("<a onclick='singleEnb_removeMMEIpInputText(this)' style='margin-left:3px'>" 
			+ "<img style='vertical-align: sub;' src='${ctx}/css/images/bi/setting_sub.png'/></a>");
	
	$mmeDiv.append($mmeSpan);
	$mmeDiv.append($mmeInput);
	$mmeDiv.append($mmeButtn);
	$mmeDiv.addClass("clearBoth");
	//将新增的输入框插入到plmn参数之前
	var plmnidDiv = $("#singleEnb_IPSECDIV");
	$mmeDiv.insertBefore(plmnidDiv); 
	
	//将当前输入框后面的图标改为删除图标，并重新绑定事件
	$(e).children("img").attr("src","${ctx}/css/images/bi/setting_add.png");
	$(e).attr("onclick", "singleEnb_addMMEIpInputText(this)");
}
//删除选中的MME地址输入框
function singleEnb_removeMMEIpInputText(e) {
	$(e).parent("div").remove();
	//如果当前第一个MME输入框的span不显示，将其显示
	var mmeDiv = $("input[name='singleEnb_LTE_SIGLINK_SERVER_LIST']:first");
	var isVisible = mmeDiv.parent("div").children("span").css("visibility");
	if (isVisible == "hidden") {
		mmeDiv.parent("div").children("span").css("visibility","visible");
	}
}
//ipsec返回div
function singleEnb_createIpsecSelectDiv(jsonOption,selectedValue,name,filedName){
	// var str = {'name1':1,"name2":2};name1为显示值，1为value
	var tip = singleEnb_createTitleForIpSec(filedName);
	var opts="<option value='null'>null</option>";
	for(var key in jsonOption){
		if(selectedValue == key){
			opts = opts+"<option value='"+key+"' selected='selected'>"+jsonOption[key]+"</option>";
		}else{
			opts = opts+"<option value='"+key+"'>"+jsonOption[key]+"</option>";
		}
	}
	var selectHtml="<select name='"+filedName+"' id='singleEnb_" + filedName +"' class='inputDivCss border border-box item'>"+opts+"</select>";
	if (!selectedValue) {
		selectedValue = "";
	}
	var div = "<div class='itemDiv'><span>"+name+"</span>"+selectHtml+"<input type='hidden' value='"+selectedValue+"' id='singleEnb_OLD_"+filedName+"'>";
	return $(div);
}
function singleEnb_authbyChange(e){
	var ele = $(e["target"]);
	var name = ele.attr("name");
	var value = ele.val();
	var oldValue = ele.nextAll("input").eq(0).val();
	var type = /^[^\d]*(\d+)$/;
	var index = type.exec(name)[1];
	if(oldValue == "cert" || oldValue == "aka_cert"){
		/* if(value == "psk" || value == "aka_psk"){
			$("input[name='ROOTCERTIFICATES_"+index+"']").parent().hide();
			$("input[name='CERTIFICATES_"+index+"']").parent().hide();
			$("input[name='PRIVATEKEYS_"+index+"']").parent().hide();
		}else{
			$("input[name='ROOTCERTIFICATES_"+index+"']").parent().show();
			$("input[name='CERTIFICATES_"+index+"']").parent().show();
			$("input[name='PRIVATEKEYS_"+index+"']").parent().show();
		} */
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
	singleEnb_changeColor(ele.attr("id"));
}
function singleEnb_validateMaxAndMinVal1(e) {
    validateMaxAndMinVal(e);
    if ($("#singleEnb_LTE_UL_DL_EARFCN").val().trim()=="") {
		$("#singleEnb_LTE_UL_DL_EARFCN").addClass("err_border");
	}
}
function singleEnb_validateIPAddress1(e){
  	var ele = $(e["target"]);
    validateIPAddress(e);
}
function singleEnb_validateIPAddress11(e){
  	var ele = $(e["target"]);
    validateIPAddress(e);
}
function singleEnb_bindInterface(e){
	var ele = $(e["target"]);
	singleEnb_changeColor(ele.attr("id"))
}
function intelFdd_validateMaxAndMinVal1(e) {
    var ele = $(e["target"]);
    if(ele.val().trim()==""){
		ele.addClass("err_border");
    }else{
	    validateMaxAndMinVal(e);
    }
}
function commonConfigSubmit(){
	var allInput = $("#singleDeviceConfigDiv input");
	for(var i=0;i<allInput.length;i++){
		if($(allInput[i]).val()==""){
			$(allInput[i]).addClass('err_border');
			$(allInput[i]).next().next().show();
		};
	}
	if ($("#selfConfigViewOrModify .item.err_border").length > 0) {
	 	$(".wirelessSetting").animate({scrollTop:0},200);
		return;
	}
	var paramMap = {};
	paramMap.LTE_TAC = $("input[name='LTE_TAC']").val();
	paramMap.LTE_CELL_IDENTITY = $("input[name='LTE_CELL_IDENTITY']").val();
	paramMap.LTE_PHY_CELLID_LIST = $("input[name='LTE_PHY_CELLID_LIST']").val();
	paramMap.LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST = $("input[name='LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST']").val();
	paramMap=JSON.stringify(paramMap);
	var serial_number = $("#singleDeviceConfig").datagrid("getSelected").serial_number;
	var params={};
	params.paramMap = paramMap;
	params.serial_number = serial_number;
	$.post("${ctx}/cell/halobSelfConfig/saveSingleCellConfigParamValue.action", params, function(data){
		if(data.success){
			$(".selfConfigSucTip").show();
			setTimeout('$(".selfConfigSucTip").fadeOut()',1000);
			$("#singleDeviceConfig").datagrid("reload");
			setTimeout('closeSelfConfig()',1000);
		}else{
			$.messager.alert("<%=rb.getString("TiShi")%>", data.message);
      	    return;
		}
	},"json");
}
function addTunnel(){
	$("#singleEnb_ipsecTunnelData_err").hide();
	if(dataList.length==0){
		$("#channel_single_1").show();
		$("#channel_single_1 input").val('');
		$("#channel_single_1 select option:first").prop('seleced','selected');
	}else if(dataList.length==2){
		$("#singleEnb_ipsecTunnelData_err").show();
	}else{
		if(dataList[0].tdId == "channel1"){
			$("#channel_single_2").show();
			$("#channel_single_2 input").val('');
			$("#channel_single_2 select option:first").prop('seleced','selected');
		}else{
			$("#channel_single_1").show();
			$("#channel_single_1 input").val('');
			$("#channel_single_1 select option:first").prop('seleced','selected');
		}
	}
	$(".wirelessSetting").animate({scrollTop:0},0);
}
function cancelTunnelModify(ele){
	$("#"+ele).hide();
}
function selfTunnelSwitchStr(value, rowData, rowIndex){
	if(value == '1'){
		value = '<%=rb.getString("KaiQi")%>';
	}else{
		value = '<%=rb.getString("GuanBi")%>';
	}
	return value;
}
</script>