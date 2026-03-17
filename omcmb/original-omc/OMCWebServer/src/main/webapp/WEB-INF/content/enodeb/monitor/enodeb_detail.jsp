<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
.infoPage {
	background:#f1f1f2;
	height:100%;
	overflow:auto;
	display:flex;
	flex-wrap:wrap;
}
.infoItem{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	padding:20px;
	width:100%;
	height:fit-content;
}
.cellInfoWarp .el-form-item .el-form-item__content {
	word-wrap: break-word;
	word-break: normal;
}
.infoItem.alarmContent {
	display:flex;
}
.alarmItem{
	border-right:1px solid #d5dcec;
	flex:20%;
	text-align:center;
}
.infoTitle {
	height:30px;
	font-size:14px;
	font-weight:bold;
}
.cellItemBoxCls {
	display:flex;
	flex-wrap:wrap;
}
.cellItemBoxCls>div {
	width:20%;
	min-width:200px;
	min-height:55px;
}
.cellItemBoxCls>div:before {
	content:attr(label);
	display:block;
	color:#7a7992;
	margin-bottom:5px;
}
.infoItem.halfItem{
	width:45%;
	min-width:400px;
	flex:1;
}
.infoItem.halfItem .cellItemBoxCls>div {
	width:30%;
	min-width:120px;
}
#enbDetailPage .cellInfoForm .el-form-item{
	width:24%;
	min-width:120px;
	margin-bottom:10px;
}
#enbDetailPage .cellInfoForm .el-form-item label {
	font-size:12px;
	color:#7a7992;
	margin-left: 0;
}
#enbDetailPage .cellInfoWarp .el-icon-arrow-right:before{
	content:"\e794";
	color:#BBB;
}
</style>
<div class="infoPage" id='enbDetailPage'>
	<div class="infoItem alarmContent">
		<div class="alarmItem">
			<div class="criticalAlarm">
				<span class="el-icon el-icon-status-alarm"></span>
				<span><%=rb.getString("JinJiGaoJing")%></span>
			</div>
			<div>{{alarmData.critical}}</div>
		</div>
		<div class="alarmItem">
			<div class="majorAlarm">
				<span class="el-icon el-icon-status-alarm"></span>
				<span><%=rb.getString("ZhuYaoGaoJing")%></span>
			</div>
			<div>{{alarmData.major}}</div>
		</div>
		<div class="alarmItem">
			<div class="minorAlarm">
				<span class="el-icon el-icon-status-alarm"></span>
				<span><%=rb.getString("CiYaoGaoJing")%></span>
			</div>
			<div>{{alarmData.minor}}</div>
		</div>
		<div class="alarmItem">
			<div class="warningAlarm">
				<span class="el-icon el-icon-status-alarm"></span>
				<span><%=rb.getString("JingGaoGaoJing")%></span>
			</div>
			<div>{{alarmData.warning}}</div>
		</div>
    </div>
	
	<div v-if="enbRow.product != 'BM'" class="infoItem enbInfo cellInfoWarp">
		<div class="infoTitle"><%=rb.getString("XiaoQuXinXi")%></div>
		<el-table v-if="enbRow.product != 'PM-B4860'" ref="cellTable" id="cellTable" :data="cellData">
			<el-table-column type="expand">
				<template slot-scope="props">
					<el-form label-position="top" inline class="cellInfoForm">
						<el-form-item label="TAC">{{props.row.tac}}</el-form-item>
						<el-form-item label="<%=rb.getString("PLMN")%>">{{props.row.plmn}}</el-form-item>
						<el-form-item label="<%=rb.getString("CPETxPower")%>">{{props.row.txPower}}</el-form-item>
						<el-form-item label="<%=rb.getString("JiZhanZhiShi")%>">{{props.row.duplexMode}}</el-form-item>
						<%-- <c:if test="${duplexMode == 'TDDMode'}"> --%>
								
						<!--<el-form-item v-if="doupleFlag == 'true'" label="<%=rb.getString("ZiZhenPeiBi")%>">{{props.row.subframeAssignment}}</el-form-item>
						<el-form-item v-if="doupleFlag == 'true'" label="<%=rb.getString("GenXuLieSuoYin")%>">{{props.row.rootSequenceIndex}}</el-form-item>
						<el-form-item v-if="doupleFlag == 'true'" label="<%=rb.getString("TeShuZiZhenPeiBi")%>">{{props.row.specialSubframePatterns}}</el-form-item>-->
						
						<el-form-item v-if="doupleFlag == 'true' && props.row.showHideCol == '1'" label="<%=rb.getString("ZiZhenPeiBi")%>">{{props.row.subframeAssignment}}</el-form-item>
						<el-form-item v-if="doupleFlag == 'true' && props.row.showHideCol == '1'" label="<%=rb.getString("GenXuLieSuoYin")%>">{{props.row.rootSequenceIndex}}</el-form-item>
						<el-form-item v-if="doupleFlag == 'true' && props.row.showHideCol == '1'" label="<%=rb.getString("TeShuZiZhenPeiBi")%>">{{props.row.specialSubframePatterns}}</el-form-item>
					</el-form>
				</template>
			</el-table-column>
			<el-table-column label="<%=rb.getString("HostName")%>" prop="cell_name"></el-table-column>
			<el-table-column label="<%=rb.getString("ShiFouJiHuo") %>" prop="activeStatus">
				<template slot-scope="scope">
					<div v-html="cellStateFormatter(scope.row.activeStatus)"></div>
				</template>
			</el-table-column>
			<el-table-column label="<%=rb.getString("ShePinKaiGuanZhuangTai")%>" prop="rfStatus">
				<template slot-scope="scope">
					<div v-html="RFStatusFormatter(scope.row.rfStatus)" style="display: flex;white-space: nowrap;"></div>
				</template>
			</el-table-column>
			<el-table-column label="PCI" prop="pci"></el-table-column>
			<el-table-column label="ECI" prop="eci"></el-table-column>
			<el-table-column label="eNB ID" prop="enb_id"></el-table-column>
			<el-table-column label="Cell ID" prop="cell_id"></el-table-column>
			<el-table-column label="<%=rb.getString("PinDian") %>" prop="earfcn">
				<template slot-scope="scope">
					<div v-html="earfcnFmt(scope.row, scope.row.earfcn, scope.$index)"></div>
				</template>
			</el-table-column>
			<el-table-column label="<%=rb.getString("DaiKuan")%>" prop="bandwidth"></el-table-column>
		</el-table>
		<el-ctable v-if="enbRow.product == 'PM-B4860'" id="boardCardTable" ref="boardCardTable" :data="boardCardCellData"
			height="300px" :pagination="false" rownumber="true" style="border:1px solid #E9E9E9;">
			<el-table-column label="<%=rb.getString("BanKaID") %>" prop="cardId"></el-table-column>
			<el-table-column label="Cell ID" prop="cellId"></el-table-column>
			<el-table-column label="<%=rb.getString("JiZhanZhiShi")%>" prop="duplexMode"></el-table-column>
			<el-table-column label="<%=rb.getString("BanKaZhuangTai") %>" prop="activeStatus">
				<template slot-scope="scope">
					<div v-html="cellStateFormatter(scope.row.activeStatus)"></div>
				</template>
			</el-table-column>
			<el-table-column label="<%=rb.getString("ShePinKaiGuanZhuangTai")%>" prop="rfStatus">
				<template slot-scope="scope">
					<div v-html="RFStatusFormatter(scope.row.rfStatus)" style="display: flex;white-space: nowrap;"></div>
				</template>
			</el-table-column>
			<el-table-column label="PCI" prop="pci"></el-table-column>
			<el-table-column label="Band" prop="band"></el-table-column>
			<el-table-column label="BandWidth" prop="bandWidth"></el-table-column>
			
		</el-ctable>
	</div>
	<div v-if="enbRow.product == 'BM'" class="infoItem enbInfo cellInfoWarp">
		<div class="infoTitle">LTE Cell Info</div>
		<el-table id="lteCellTable" ref="lteCellTable" :data="lteCellInfoTableData">
			<el-table-column type="expand">
				<template slot-scope="props">
					<el-form label-position="top" inline class="cellInfoForm">
						<el-form-item label='<%=rb.getString("PinLvHeZi")%>'>
							<div>{{ props.row.eARFCNDL ? earfcnToFrequency(props.row.eARFCNDL.split(',').map(item => earfcnFormatter(item, props.row, props.$index))) : '' }}</div>
						</el-form-item>
						<el-form-item label='<%=rb.getString("CPETxPower")%>'>
							<div v-if="props.row.antennaPortsCount && props.row.xCOMMaxTxPowerExpanded">{{props.row.antennaPortsCount}}*{{props.row.xCOMMaxTxPowerExpanded}}dBm</div>
							<div v-else></div>
						</el-form-item>
						<el-form-item label="CPRI ID">{{props.row.index}}</el-form-item>
						<el-form-item label='<%=rb.getString("LuYouSuoYin")%>'>{{props.row.lteCellWithRuList}}</el-form-item>
						<el-form-item label="Band">{{props.row.freqBandIndicator}}</el-form-item>
						<el-form-item label='<%=rb.getString("ChuanShuGongLv")%>' v-if="false">
							<div v-if="props.row.antennaPortsCount && props.row.xCOMMaxTxPowerExpanded">{{props.row.antennaPortsCount}}*{{props.row.xCOMMaxTxPowerExpanded}}dBm</div>
							<div v-else></div>
						</el-form-item>
						<el-form-item label="PLMN">{{props.row.existPlmnidList}}</el-form-item>
					</el-form>
				</template>
			</el-table-column>

			<el-table-column label="Index" prop="index"></el-table-column>
			<el-table-column label='<%=rb.getString("ShiFouJiHuo") %>' prop="opState">
				<template slot-scope="scope">
					<div v-if="scope.row.opState == '0'" style="color:#FF4614;"><%= rb.getString("QuJiHuo")%></div>
					<div v-else-if="scope.row.opState == '1'"><%= rb.getString("JiHuo")%></div>
					<div v-else></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ShePinKaiGuanZhuangTai")%>' prop="xCOMRadioEnable">
				<template slot-scope="scope">
					<div v-if="scope.row.xCOMRadioEnable == '0'" style="color:#FF4614;"><%= rb.getString("Guan")%></div>
					<div v-else-if="scope.row.xCOMRadioEnable == '1'"><%= rb.getString("Kai")%></div>
					<div v-else></div>
				</template>
			</el-table-column>
			<el-table-column label="PCI" prop="phyCellID"></el-table-column>
			<el-table-column label="ECI" prop="eci"></el-table-column>
			<el-table-column label='<%=rb.getString("EnodebId")%>' prop="enbId"></el-table-column>
			<el-table-column label='<%=rb.getString("XIAOQUID")%>' prop="cellId"></el-table-column>
			<el-table-column label="<%=rb.getString("PinDian") %>" prop="eARFCNDL">
				<template slot-scope="scope">
					<div v-html="earfcnFmt(scope.row, scope.row.eARFCNDL, scope.$index)"></div>
				</template>
			</el-table-column>
			<el-table-column label="<%=rb.getString("DaiKuan")%>" prop="dLBandwidth" :formatter="bandwidthFmtOne"> </el-table-column>
		</el-table>
	</div>
	<div v-if="enbRow.product == 'BM'" class="infoItem enbInfo cellInfoWarp">
		<div class="infoTitle">GSM Cell Info</div>
		<el-table id="gsmCellTable" ref="gsmCellTable" :data="gsmCellInfoTableData">
			<el-table-column type="expand">
				<template slot-scope="props">
					<el-form label-position="top" inline class="cellInfoForm">
						<el-form-item label='<%=rb.getString("JiZhanShiBieMa")%>'>{{props.row.gsmBSIC}}</el-form-item>
						<el-form-item label="CPRI ID">{{props.row.index}}</el-form-item>
						<el-form-item label='<%=rb.getString("LuYouSuoYin")%>'>{{props.row.gsmCellWithRuRelation}}</el-form-item>
						<el-form-item label="LAC" v-if="false">{{props.row.currLocAreaCode}}</el-form-item>
						<el-form-item label="ARFCN">{{props.row.currentArfcn}}</el-form-item>
						<el-form-item label='<%=rb.getString("PinLvHeZi")%>'>
							<div v-if="props.row.currentArfcn !== undefined && props.row.currentArfcn !== null && computeGsmFreq(props.row.currentArfcn) && computeGsmFreq(props.row.currentArfcn).uplink && computeGsmFreq(props.row.currentArfcn).downlink">
								Uplink: {{ computeGsmFreq(props.row.currentArfcn).uplink }} / Downlink: {{ computeGsmFreq(props.row.currentArfcn).downlink }}
							</div>
							<div v-else></div>
						</el-form-item>
						<el-form-item label='<%=rb.getString("ChuanShuGongLv")%>'>
							<div v-if="props.row.gsmBtsRFPower">{{props.row.gsmBtsRFPower}}dBm</div>
							<div v-else></div>
						</el-form-item>
						<el-form-item label="ipa">
							<div>{{ props.row.iPAUnitID ? props.row.iPAUnitID.split('-')[0] : '' }}</div>
						</el-form-item>
						<el-form-item label='<%=rb.getString("DanYuanID")%>'>
							<div>{{ props.row.iPAUnitID && props.row.iPAUnitID.includes('-') ? props.row.iPAUnitID.split('-')[1] : '' }}</div>
						</el-form-item>
						<el-form-item label='<%=rb.getString("DuiDuanIP")%>'>{{props.row.oMLRemoteIP}}</el-form-item>
						<el-form-item label='<%=rb.getString("YuanChengIPBeiFei")%>'>{{props.row.oMLRemoteIPBak}}</el-form-item>
						<el-form-item label="CellDt">
							<span v-if="props.row.gsmMACTtiSwitch == '1'"><%= rb.getString("Kai")%></span>
							<span v-else-if="props.row.gsmMACTtiSwitch == '0'"><%= rb.getString("Guan")%></span>
							<span v-else></span>
						</el-form-item>
						<el-form-item label='<%=rb.getString("GenZongXiaoXiLeiXing")%>'>
							<span v-if="props.row.gsmMACTraceMsgType == '1'"><%=rb.getString("SheZhiTTIGenZongBianHao")%></span>
							<span v-else-if="props.row.gsmMACTraceMsgType == '3'"><%=rb.getString("BuHuoIQRiZhi")%></span>
							<span v-else-if="props.row.gsmMACTraceMsgType == '9'"><%=rb.getString("XianShiTTIGenZongBianHao")%></span>
							<span v-else></span>
						</el-form-item>
						<el-form-item label='<%=rb.getString("CunGenZhi")%>'>{{props.row.gsmMACStubVal}}</el-form-item>
						<el-form-item label='<%=rb.getString("GenZongBianHao")%>'>{{props.row.gsmMACTraceNum}}</el-form-item>
					</el-form>
				</template>
			</el-table-column>
			<el-table-column label="Index" prop="index"></el-table-column>
			<el-table-column label='<%=rb.getString("ShiFouJiHuo") %>' prop="opState">
				<template slot-scope="scope">
					<div v-if="scope.row.opState == '0'" style="color:#FF4614;"><%= rb.getString("QuJiHuo")%></div>
					<div v-else-if="scope.row.opState == '1'"><%= rb.getString("JiHuo")%></div>
					<div v-else></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ShePinKaiGuanZhuangTai")%>' prop="rfState">
				<template slot-scope="scope">
					<div v-if="scope.row.rfState == '0'" style="color:#FF4614;"><%= rb.getString("Guan")%></div>
					<div v-else-if="scope.row.rfState == '1'"><%= rb.getString("Kai")%></div>
					<div v-else></div>
				</template>
			</el-table-column>
			<el-table-column label="Abis Link Status" prop="bscSelect">
				<template slot-scope="scope">
					<div v-if="scope.row.bscSelect == '255'"><%=rb.getString("IpsecWeiLianJie")%></div>
					<div v-else-if="scope.row.bscSelect == '0' || scope.row.bscSelect == '1'"><%= rb.getString("LianJieZhengChang")%></div>
					<div v-else></div>
				</template>
			</el-table-column>
			<el-table-column label="UE Connections" prop="ueCount"></el-table-column>
			<el-table-column label="Cell ID" prop="gsmCellID"></el-table-column>
			<el-table-column label="LAC" prop="currLocAreaCode"></el-table-column>
			<el-table-column label='PLMN' prop="networkCountryCode">
				<template slot-scope="scope">
					<div v-if="scope.row.networkCountryCode && scope.row.mobileNetworkCode">{{scope.row.networkCountryCode}}{{scope.row.mobileNetworkCode}}</div>
					<div v-else></div>
				</template>
			</el-table-column> 
		</el-table>
	</div>

	<div class="infoItem enbInfo cellInfoWarp" v-if="enbRow.product_name == 'Nova430X' || enbRow.product_name == 'Neutrino430X'">
		<div class="infoTitle">WiFi Info</div>
		<el-ctable id="wifiTable" ref="wifiTable" :url="wifiTableUrl" :time="6"
			height="200px" :pagination="true" rownumber="true" style="border:1px solid #E9E9E9;">
			<el-table-column label="<%=rb.getString("Title_SheBeiBianMa") %>" prop="serialNumber"></el-table-column>
			<el-table-column label="<%=rb.getString("Title_SheBeiMingCheng") %>" prop="deviceName"></el-table-column>
			<el-table-column label="MAC" prop="macAddress"></el-table-column>
			<el-table-column label="<%=rb.getString("IPDiZhi")%>" prop="ipAddress"></el-table-column>
			<el-table-column label="<%=rb.getString("ChanPinXingHao")%>" prop="productModel"></el-table-column>
			<el-table-column label="<%=rb.getString("SoftwareVersion")%>" prop="softwareVersion"></el-table-column>
		</el-ctable>
		
	</div>

	<div class="infoItem enbInfo">
		<div class="infoTitle"><%=rb.getString("SASSheBeiXinXi")%></div>
		<div class="cellItemBoxCls">
			<div class="cell-item-cls" label="<%=rb.getString("JiZhanXuLieHao")%>">${serialNumber}</div>
			<div class="cell-item-cls" label="<%=rb.getString("YunXingShiJian")%>">${systemUpTime}</div>
			<div class="cell-item-cls" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>">${productType}</div>
			<div class="cell-item-cls" label="<%=rb.getString("SheBeiXingHaoMing")%>" v-html="modelType"></div>
			<div class="cell-item-cls" label="<%=rb.getString("DiYiCiLianJieShiJian")%>">${firstPeriodTime}</div>
			<div class="cell-item-cls" label="<%=rb.getString("ShangCiLianJieShiJian")%>">${lastPeriodTime}</div>
			<div class="cell-item-cls" label="<%=rb.getString("SoftwareVersion")%>">${softwareVersion}</div>
			<div class="cell-item-cls" label="<%=rb.getString("MACDiZhi")%>">${macAddress}</div>
			<div class="cell-item-cls" label="<%=rb.getString("FirmwareVersion")%>">${hardwareVersion}</div>
			<div class="cell-item-cls" label="<%=rb.getString("GPSBanBen")%>">${GPSVersion}</div>
			<div class="cell-item-cls" label="<%=rb.getString("SheBeiZu")%>">${groupName}</div>
			<div v-if="ShopIdShow" class="cell-item-cls vertic" :label="siteIdLabel">${siteID}</div>
            <div class="cell-item-cls" :label="currentRemarkLabel">${remark}</div>
        </div>
	</div>
	<div class="infoItem enbInfo">
		<div class="infoTitle"><%=rb.getString("ZhuangTai")%></div>
		<div class="cellItemBoxCls">
			<div v-show="sasShow" label="CBSD <%=rb.getString("ZhuangTai")%>" v-html="sasStatus"></div>
			<div label="<%=rb.getString("MMEZhuangTai")%>">
				<div v-if="['','NULL','null',null,undefined].includes(enbRow.NEW_MME_STATUS)" style="display: flex;align-items: center;">
					<span v-html="oldMmeStatusFmt(enbRow, enbRow.mme_status, 'left')" ></span>
					<el-popover title="All MME" popper-class="mmePopoverClass" v-if="oldMmeStatusFmt(enbRow, enbRow.mme_status, 'list').length > 0">
						<span style="color:#4d84ff;cursor:pointer;" slot="reference" v-if="oldMmeStatusFmt(enbRow,enbRow.mme_status, 'list').length > 0">
							[ <span v-html="oldMmeStatusFmt(enbRow, enbRow.mme_status, 'right')"></span> ]
						</span>
						<div class="mme-list">
							<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
							<div class="mme-list-item" v-for="item in oldMmeStatusFmt(enbRow, enbRow.mme_status, 'list')">
								<span v-if="item.status == '1'" style="margin-top:10px;" class="el-icon el-icon-status-MME greenIcon"></span>
								<span v-if="item.status == '0'" style="margin-top:10px;" class="el-icon el-icon-status-MME redIcon"></span> 
								<div class="mme-info">
									<span>MME IP : {{item.mmeIp}}</span>
									<span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEWeiLianJie")%></span>
									<span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEYiLianJie")%></span>
									<span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
									<span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
								</div>
							</div>
						</div>
					</el-popover>
				</div>
				<div v-if="!['','NULL','null',null,undefined].includes(enbRow.NEW_MME_STATUS)" style="display: flex;align-items: center;">
					<span v-html="mmesStatusFmt(enbRow.NEW_MME_STATUS)" ></span>
					<el-popover title="All MME" popper-class="mmePopoverClass">
						<span style="color:#4d84ff;" slot="reference">
							[ <span v-html="parseMME(enbRow.NEW_MME_STATUS).length"></span> 
							<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
						</span>
						<div class="mme-list" v-if="!['QAFA','QATA'].includes(enbRow.product)">
							<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
							<div class="mme-list-item" v-for="item in parseMME(enbRow.NEW_MME_STATUS)">
								<span v-if="item.status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
								<span v-if="item.status == '0'" class="el-icon el-icon-status-MME redIcon"></span>
								<div class="mme-info">
									<span>MME IP: {{item.mmeIp}}</span>
									<span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%>: Disconnected</span>
									<span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%>: Connected</span>
									<span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%>: --</span>
									<span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%>: </span>
									<span><%=rb.getString("PLMN")%>: {{item.plmnId}}</span>
								</div>
							</div>
						</div>
						<div class="mme-list" v-if="['QAFA','QATA'].includes(enbRow.product)">
							<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
							<div class="mme-list-item" v-for="item in parseMME(enbRow.NEW_MME_STATUS)">
								<span v-if="item.status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
								<span v-if="item.status == '0'" class="el-icon el-icon-status-MME redIcon"></span> 
								<div class="mme-info">
									<span>MME IP : {{item.mmeIp}}</span>
									<span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : Disconnected</span>
									<span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : Connected</span>
									<span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
									<span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
									<span>MME Port : {{item.mmePort}}</span>
								</div>
							</div>
						</div>
					</el-popover>
				</div>
			</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("KPIShangBaoZhuangTai")%>" v-html="pmStatus"></div>
			<div class="cell-item-cls vertic" label="HaloX">
				<div v-if="enbRow.form == '0'|| enbRow.form == '1' || enbRow.form == '2'">
					HaloD
					<el-popover title="" popper-class="halodCellPopoverClass" trigger="hover" width='240' @show="halodCellPopoverShow" @hide="halodCellPopoverHide">
						<span class="haloDModeCodeCls" slot="reference">{{enbRow.form}}</span>
						<div style="padding:10px;border-top:1px solid #E9E9E9;min-height:120px;" v-loading="halodCellPopoverLoading" >
							<div style="padding:10px;border:1px dashed #E9E9E9;">
								<div v-for="item in halodCellDataList" style="height:30px;display:flex;align-items: center;">
									<span class="itemValueCls">{{item.serialNumber}}</span>
									<span class="haloDModeCodeCls">{{item.form}}</span>
								</div>
							</div>
						</div>
					</el-popover>
				
				</div>
				<div v-else>
					<div v-if="enbRow.halob_flag == '1'" style="display: flex;align-items: center;">
						HaloB<span class='el-icon el-icon-status-enable' style='margin-left:3px;font-size: 20px;'></span>
					</div>
					<div v-else-if="enbRow.halob_flag == '0'" style="display: flex;align-items: center;">
						HaloB<span class='el-icon el-icon-status-disable' style='margin-left:3px;font-size: 20px;'></span>
					</div>
					<div v-else>--</div>
				</div>
			</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("TongBuZhuangTai")%>" v-html="syncStatus"></div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("YouXiaoQi")%>" v-show="isExpiredShow">${expiryDate}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("SuoDingZhuangTai")%>" v-show="isExpiredShow" v-html="lockStatus"></div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("UEShu")%>" >${UECount}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("CPELianJieShu")%>" v-html="cpeCountFat" ></div>
		</div>
	</div>
	
	<div class="infoItem enbInfo halfItem">
		<div class="infoTitle"><%=rb.getString("WeiZhi")%></div>
		<div class="cellItemBoxCls">
			<div class="cell-item-cls vertic" label="<%=rb.getString("JingDu")%>">${gpsLongitude}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("WeiDu")%>">${gpsLatitude}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("GaoDu")%>">${gpsHeight}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("JiXieXiaQingJiao")%>">${mechanical_downtilt}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("DianZiXiaQingJiao")%>">${electronic_downtilt}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("ChuiZhiBoSuKuanDu")%>">${vertical_3dB_beam_width}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("ShuiPinFangWeiJiao")%>">${horizontal_azimuth}</div>
		</div>
	</div>
	<div class="infoItem enbInfo halfItem">
		<div class="infoTitle"><%=rb.getString("WangLuoSheZhi")%></div>
		<div class="cellItemBoxCls">
			<div class="cell-item-cls vertic" v-if="isSuperAdmin" label="MME Interface Binding">${mmeInterfaceBinging}</div>
			<div class="cell-item-cls vertic" v-if="isSuperAdmin" label="<%=rb.getString("MMEPoolIPSECDiZhi")%>">${ipsecAddress}</div>
			<div class="cell-item-cls vertic" label="<%=rb.getString("IPDiZhi")%>" v-html="ipAddress">${ipAddress}</div>
		</div>
	</div>
	<div class="infoItem enbInfo" style="position: relative;" v-if="['QAFA','QAFB','QATA','BAIBLQ'].includes(enbRow.product)">
		<div class="infoTitle"><%=rb.getString("LinQu")%></div>
		<!-- 按钮  同步 -->
		<div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:30px;top:10px;" @click="syncAnrCellClick" tip="<%=rb.getString("TongBu")%>">
			<span class="el-icon el-icon-circle-refresh"></span>
		</div>
		<el-ctable id="adjacentRegionTable" ref="adjacentRegionTable" :url="adjacentRegionTableUrl"
			height="180px" :time="6" :pagination="false" :rownumber="false" style="border:1px solid #E9E9E9;">
			<el-table-column label="<%=rb.getString("XuLieHao") %>" prop="index" width="60"></el-table-column>
			<el-table-column label="<%=rb.getString("PLMN")%>" prop="PLMN" min-width="60"></el-table-column>
			<el-table-column label="ECI(ECI=eNB_ID*256+Cell ID)" prop="ECI" min-width="120"></el-table-column>
			<el-table-column label="EARFCN" prop="EARFCN" min-width="40"></el-table-column>
			<el-table-column label="PCI" prop="PCI" min-width="40"></el-table-column>
			<el-table-column label="TAC" prop="TAC" min-width="40"></el-table-column>
			<el-table-column label="csg-ID" prop="csg-ID" min-width="80"></el-table-column>
			<el-table-column label="CellType" prop="CellType" min-width="80"></el-table-column>
		</el-ctable>
	</div>
	
</div>
<script type="text/javascript">
var enbStatusTimer;
var timeParam = getNowTimeToZoneTimeRange(timeZone, 144)
var start_time_enb = timeParam.start_time.substring(0,11)+"00:00:00";
var end_time_enb = timeParam.end_time;
var enbDetailVue = new Vue({
	el:'#enbDetailPage',
	data(){
		var vm = this;
		return {
            enbSelectedRow:{},
			alarmData:{
				minor:'0',
				major:'0',
				critical:'0',
				warning:'0'
			},
			cellData:[],
			doupleFlag:'${duplexMode == "TDDMode"}',
			height:'100%',
			boardCardCellData:[],
			lteCellInfoTableData:[],	
			gsmCellInfoTableData:[],
			queryCellParams:{
				smallCellCode : "${smallCellCode}"
			},
			adjacentRegionTableUrl:'${ctx}/cell/param/getNeigborCellStateDataList.action?smallCellCode=${smallCellCode}',
			
            halodCellPopoverLoading:false,
			halodCellDataList:[],

			actionAble: {
				sasFlag: true,
				settingFlag: true,
				syncFlag: true,
				rebootFlag: true,
				resetConfigFlag: true
			},

            wifiTableUrl:'${ctx}/cell/CPE/queryCpeInfosByEnbSn.action?serial_number=${serialNumber}',
            currentRemarkLabel: '',
		}
	},
    computed: {
    	isCA() {
    		return this.enbSelectedRow.ca_flag == 1;
    	},
		enbRow() { // 当前设备数据
			return this.enbSelectedRow;
		},
		isSuperAdmin() { //是否是超级管理员
			return is_super_user == 'true';
		},
		ipAddress() {
			return ipAddrFormatter('${ipAddress}');
		},
		modelType() { // 设备型号格式化
			var value = '${modultType}',
				capablity = '${capablity}';
			if(capablity == 'enable' && isLWAEnable){
				value = "<span class='cpeLwaKai' style='font-size:18px'>"+"</span>"+ "<span style='font-size:12px;padding-top: 5px;'>"+(value)+"</span>";
			}else if(capablity =='disable' && isLWAEnable){
				value = "<span class='cpeLwaWu' style='font-size:18px'>"+"</span>"+ "<span style='font-size:12px;padding-top: 5px;'>"+(value)+"</span>";
			}
			return value;
		},
		activeStatus() { // 激活状态格式化
			var status = '${activeStatus}';
			if([1,0,'1','0','',null].includes(status)) {
				return cellStateFormatter(status);
			}else {
				return this.activeMultFmt({},status);
			}
		},
		rfStatus() {
			var status = '${rfStatus}',
				list = status.split(','),
				html = '';
			var flag = true;
			list.map(function(item){
				if(flag){
					html = RFStatusFormatter(item);
					flag = false;
				}
			});
			return html;
		},
		pmStatus() {
			return kpiStatusFormatter('${pmReportStatus}');
		},
		syncStatus() {
			var vm = this,
				val = '',
				value = '${synStatus}';
				
			if (value == null) {
				return null;
			} else if (value == "--" ){
				return "--";
			} else if (value == ("GPS " +'<%= rb.getString("ZhengZaiTongBu")%>')) {
				val ="GPS "+ '<%= rb.getString("ZhengZaiTongBu")%>';
				value = "<span>"+(val)+"</span>"
			}else if (value == ("1588 " +'<%= rb.getString("ZhengZaiTongBu")%>')) {
				val ="1588 "+ '<%= rb.getString("ZhengZaiTongBu")%>';
				value = "<span>"+(val)+"</span>"
			}else if (value == ("REM " +'<%= rb.getString("ZhengZaiTongBu")%>')) {
				val ="REM "+ '<%= rb.getString("ZhengZaiTongBu")%>';
				value = "<span>"+(val)+"</span>"
			}else if (value == "GPS "+ '<%= rb.getString("TongBuChengGong")%>' ) {
				val ="GPS "+ '<%= rb.getString("TongBuChengGong")%>';
				value = "<span>"+(val)+"</span>"
			}else if (value == "1588 "+'<%= rb.getString("TongBuChengGong")%>' ) {	
				val = '<%= rb.getString("TongBuChengGong")%>';
				val = "1588 " + val;
				value = "<span>"+(val)+"</span>"
			}else if (value == "REM "+'<%= rb.getString("TongBuChengGong")%>') {	
				val = '<%= rb.getString("TongBuChengGong")%>';
				val = "REM " + val;
				value = "<span>"+(val)+"</span>"
			}else if (value == '<%= rb.getString("WeiTongBu")%>') {	
				val = '<%= rb.getString("WeiTongBu")%>';
				value = "<span class='offStatusCls'>"+(val)+"</span>"
			}
			return '<div style="min-width: 130px;diplay: line-block;">'+value+'</div>';
		},
		lockStatus() {
			return lockStatusFmt('${lockStatus}');
		},
		isExpiredShow() {
			return writableMap["CODE_ENB_EXPIRY_DATE"] != undefined;
		},
		cellName() {
			return unescape('${cellName.replaceAll("\'","%27").replaceAll("\"","%22")}');
		},
		sasShow() {
			return  writableMap['CODE_ADVANCE_SAS'] != undefined;
		},
		sasStatus() {
			var value = '${sasState}',
				sasObj = {
					"0":"Unregistered",
					"1":"Registered",
					"3":"Granted",
					"4":"Grant Suspended",
					"5":"Authorized",
					"6":"Transmission"
				};
			return sasObj[value];
		},
		cpeCountFat(){
			var rowData = this.enbSelectedRow;
				value = '${cpe_connect}';
			if(value == -1 || value == null){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>--</a>"; 
            }else if(value === 0 || value === '0') {
                return value;
            }else{
                var is436Q = ['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC'].includes(rowData.platformType);
                
                return "<a style='color:#1DA3FC;text-decoration:underline' href='#' onclick='getueCpeCountsData(&quot;" 
                        + rowData.small_cell_code + "&quot;,&quot;" + rowData.CELL_IDENTITY + "&quot;,&quot;" + rowData.PHYCELLID 
                        + "&quot;,&quot;" + rowData.EARFCNDLINUSE + "&quot;,&quot;" + rowData.serial_number + "&quot,&quot;" 
                        + rowData.host_name + "&quot,\"enodeb_cpe_connect\","+is436Q+")'>"+value+"</a>"; 
            }
		},
		isCell2Show(){
			return "${platformTypeFlag}" == '1' ? true : false;
		},
		isCell3Show(){
			return "${platformTypeFlag}" == '3' ? true : false;
		},
		ShopIdShow(){
			return siteIdShow == 'true' ? true : false;
		},
		siteIdLabel(){
			return siteIdLabel
		},
		siteNameLabel(){
			return siteNameLabel
		},
		optBtnShow() {
			return writableMap['CODE_ENB_MONITOR'] == true;
		},
	},
	watch:{},
	methods:{
		// 初始化
		init(row,code,sn){
			var vm = this;
			var cellParams1 = {
					'showHideCol':'1',
					"cell_name":"${cellName}",
					"activeStatus":"${activeStatus}",
					"rfStatus":"${rfStatus}",
					"pci":"${pci}",
					"eci":"${eci}",
					"enb_id":"${enbId}",
					"cell_id":"${cellId}",
					"earfcn":"${earfcn}",
					"bandwidth":"${bandwidth}",
					
					"tac":"${tac}",
					"plmn":"${plmn}",
					"txPower":"${tx_power}",
					"duplexMode":"${duplexMode}",
					"subframeAssignment":"${subframeAssignment}",
					"rootSequenceIndex":"${rootSequenceIndex}",
					"specialSubframePatterns":"${specialSubframePatterns}",
				};
			vm.cellData.push(cellParams1);
			
			var cellParams2 = {
					"cell_name":"${cellName_cell2}",
					"activeStatus":"${activeStatus_cell2}",
					"rfStatus":"${rfStatus_cell2}",
					"pci":"${pci_cell2}",
					"eci":"${eci_cell2}",
					"enb_id":"${enbId2}",
					"cell_id":"${cellId2}",
					"earfcn":"${earfcn_cell2}",
					"bandwidth":"${bandwidth_cell2}",
					
					"tac":"${tac_cell2}",
					"plmn":"${plmn_cell2}",
					"txPower":"${tx_power_cell2}",
					"duplexMode":"${duplexMode}",
				};
			var cellParams3 = {
					'showHideCol':'1',
					"cell_name":"${cellName_cell3}",
					"activeStatus":"${activeStatus_cell3}",
					"rfStatus":"${rfStatus_cell3}",
					"pci":"${pci_cell3}",
					"eci":"${eci_cell3}",
					"enb_id":"${enbId3}",
					"cell_id":"${cellId3}",
					"earfcn":"${earfcn_cell3}",
					"bandwidth":"${bandwidth_cell3}",
					
					"tac":"${tac_cell3}",
					"plmn":"${plmn_cell3}",
					"txPower":"${tx_power_cell3}",
					"duplexMode":"${duplexMode}",
				};
            vm.enbSelectedRow = row;
			if(!vm.isCA){
				cellParams2.subframeAssignment = "${subframeAssignment_cell2}";
				cellParams2.rootSequenceIndex = "${rootSequenceIndex_cell2}";
				cellParams2.specialSubframePatterns = "${specialSubframePatterns_cell2}";
				
				cellParams3.subframeAssignment = "${subframeAssignment_cell3}";
				cellParams3.rootSequenceIndex = "${rootSequenceIndex_cell3}";
				cellParams3.specialSubframePatterns = "${specialSubframePatterns_cell3}";
				cellParams2.showHideCol = '1';
			}else{
				//不显示了字段名，也不不显示值
				cellParams2.showHideCol = '0';
			}

			if(vm.isCell2Show){
				vm.cellData.push(cellParams2)
			}
			if(vm.isCell3Show){
				vm.cellData.push(cellParams2,cellParams3);
			}

			vm.initAlarmCount();
			if('${productType}' == 'BM'){
				vm.getLteOrGsmCellInfos();
			}
		
			if('${productType}' == 'PM-B4860'){
				vm.getCellInfos();
			}
			
            vm.getCustomLabelData();

		},
		// 获取自定义label信息
        getCustomLabelData() {
            var vm = this;

            axios.post('${ctx}/cell/columnAlias/queryColumnAliasConfigs.action').then(function(response){
                var data = response.data || [];

                data.forEach(function(item){
                    if(item.columnName == 'remark'){
                        vm.currentRemarkLabel = item.columnAlias || 'Remark';
                    }
                });
            }).catch(function(error){});
        },
        earfcnFmt(row, value, index) {
            if(value) {
                var resList = [];
                value.split(',').map(function(item){
                    resList.push(earfcnFormatter(item, row, index));
                });
                
                return resList.join(',');
            }else {
                return '';
            }
        },
		earfcnToFrequency(freValue){
			var vm = this;
				frequencyValues = '';
			//获取freValue 值中的小括号中的值
			
			if (freValue && Array.isArray(freValue)) {
				var freqArray = [];
				freValue.forEach(function(item) {
					// 从每个数组元素中提取小括号中的频率值
					var match = item.match(/\(([^)]+)\)/);
					if (match) {
						// 提取括号中的内容，去掉 "MHz" 后缀
						var freq = match[1].replace('MHz', '');
						freqArray.push(freq);
					}
				});
				// 将频率值数组转换为逗号分隔的字符串
				frequencyValues = freqArray.join(',');
			}
			return frequencyValues;
		},
		
		// 获取基站告警统计
		initAlarmCount(){
			var vm = this,
				url='${ctx}/fault/view/queryViewAlarmCount.action',
				params = {
					timeZone: timeZone,
					neType:'ENB',
					alarmType:'ACTIVE',
					deviceCode : "${smallCellCode}"
				};
			axios.post(url,stringify(params)).then(function(response){
				let data = response.data
				if(data){
					Object.assign(vm.alarmData,data)
				}
			}).catch(function(error){})
		},
		// 获取板卡小区参数
		getCellInfos(){
			var vm = this,
				url='${ctx}/pm/nxp/getSlotCellInfos.action',
				params = {
					smallCellCode : "${smallCellCode}"
				};
			axios.post(url,stringify(params)).then(function(response){
				let data = response.data
				if(data){
					vm.boardCardCellData = data;
				}
			}).catch(function(error){})
		},
		// GSM ARFCN -> 上/下行频率计算 (公式参照后端 Java)
        computeGsmFreq(arfcn){
        	let res = { uplink: '', downlink: '' };
        	if(arfcn === undefined || arfcn === null || arfcn === ''){ return res; }
        	// 可能传入字符串，先转整数
        	let n = parseInt(arfcn,10);
        	if(isNaN(n)){ return res; }
            // 公式：
        	// 0 <= n <= 125:
        	//   UL = (8900 + 2*n)/10  MHz
        	//   DL = (9350 + 2*n)/10  MHz
        	// 975 <= n <= 1023:
        	//   UL = (8800 + 2*(n-974))/10 MHz
        	//   DL = (9250 + 2*(n-974))/10 MHz
        	function fmt(v){ return (Math.round(v*10)/10); }
        	if(n >=0 && n <=125){
        		res.uplink = fmt((8900 + 2*n)/10.0);
        		res.downlink = fmt((9350 + 2*n)/10.0);
        	}else if(n >= 975 && n <= 1023){
        		res.uplink = fmt((8800 + 2*(n-974))/10.0);
        		res.downlink = fmt((9250 + 2*(n-974))/10.0);
        	}
        	return res;
        },
       
		getLteOrGsmCellInfos(){
			var vm = this,
				url='${ctx}/cell/cpeinfos/queryOverviewCellInfo.action',
				params = {
					smallCellCode : "${smallCellCode}"
				};
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data;
				
				if(!isEmptyObject(data)){
					if(data.lte){
						vm.lteCellInfoTableData = data.lte; 
                    }
					if(data.gsm){
						vm.gsmCellInfoTableData = data.gsm;
					}	
				}
			}).catch(function(error){})
		},
		bandwidthFmtOne(row,column,value,index) {
			var vm = this,
				rels = {
					'25': '5MHz',
					'50': '10MHz',
					'75': '15MHz',
					'100': '20MHz'
				};
			return rels[value]||'';
		},
		// 旧mme状态 格式化
		oldMmeStatusFmt(rowData,value,type){
			var vm = this,
				mmeStatus,textVal,value,hasDisconn=false,hasConn=false,mmeDataList=[],leftStr='',parseMMEOnNum=0;
		
			if (value == null || value == "") {
				return type == 'list' ? [] : null;
			}
			
			if (value == "1") {
				mmeDataList.push({mmeIp:rowData.s1siglinkserverlist,status:'1'})
				mmeStatus ='';
				textVal = '<%= rb.getString("MMEYiLianJie")%>'
			} else if (value == "0") {
				mmeDataList.push({mmeIp:rowData.s1siglinkserverlist,status:'0'})
				mmeStatus ='offStatusCls';
				textVal = '<%= rb.getString("MMEWeiLianJie")%>'
			} else if (value == "2") {
				mmeStatus ='';
				textVal = '--'
			} else{
				var mmePools = value ? value.split(",") : [];
				for(var i = 0; i < mmePools.length; i++ ){
					var mme = mmePools[i];
					var mmeArr = mme ? mme.split("=") : [];
					
					if(mmeArr.length == 2 && "mme1" == mmeArr[0]){
						mmeDataList.push({mmeIp:rowData.mme_pool_1,status:mmeArr[1]})
					}else if(mmeArr.length == 2 && "mme2" == mmeArr[0]){
						mmeDataList.push({mmeIp:rowData.mme_pool_2,status:mmeArr[1]})
					}
				}
				mmeDataList.map((item)=>{
					if (item.status == '1'){
						parseMMEOnNum += 1;
						hasConn = true;
					}else if(item.status == '0'){
						hasDisconn = true
					}
				})
				if (hasDisconn == true){
					if (hasConn == false) {
						mmeStatus ='offStatusCls';
						textVal = '<%= rb.getString("MMEWeiLianJie")%>'
					}else{
						mmeStatus ='offOrOnStatusCls';
						textVal = '<%= rb.getString("MMEYiLianJie")%>'
					}
				}else {
					mmeStatus ='';
					textVal = '<%= rb.getString("MMEYiLianJie")%>'
				}
			}
			mmeDataList.map((item)=>{
				if (item.status == '1'){
					parseMMEOnNum += 1;
				}
			})
			if(type == 'left'){
				value = "<span class='"+ mmeStatus +"'>" + textVal +"</span>"
			}else if(type == 'right'){
				value = parseMMEOnNum + '/' + mmeDataList.length
			}else if(type == 'list'){
				value = mmeDataList
			}
			return value;
		},
		// mme状态 格式化
		mmesStatusFmt(str){
			var mmelist = eval('('+str+')');
			var mmeStatus,textVal,value,hasDisconn=false,hasConn=false;
			mmelist.map(function(item){
				if (item.status == '1'){
					hasConn = true;
				}else if(item.status == '0'){
					hasDisconn = true
				}
			})
			
			if (hasDisconn == true){
				if (hasConn == false) {
					mmeStatus ='offStatusCls';
					textVal = 'Disconnected'
				}else{
					mmeStatus ='offOrOnStatusCls';
					textVal = 'Connected'
				}
			}else {
				mmeStatus ='';
				textVal = 'Connected'
			}
			
			value = "<span class='"+ mmeStatus +"'>" + textVal +"</span>"
			return value;
		},
		// mme 格式化分割字符串
		parseMME(str) {
			if(str) {
				try {
					return eval('('+str+')');
				} catch(e) {
					return [];
				}
			}else {
				return [];
			}
		},
		halodCellPopoverShow(){ // HaloD cell信息Popover 打开事件
			var vm = this;
			vm.halodCellPopoverLoading = true;
			axios.post('${ctx}/cell/halod/queryHalodRelationInfo.action',stringify({serialNumber:vm.enbRow.serial_number})).then(function(response){
				let data = response.data
				if(data){
					vm.halodCellDataList = data.snList;
					vm.halodCellPopoverLoading = false;
				}
			}).catch(function(error){})
		},
		halodCellPopoverHide(){ // HaloD cell信息Popover 关闭事件
			var vm = this;
			vm.halodCellDataList = [];
		},
		// 同步 ANR邻区邻频
		syncAnrCellClick(){
			var vm = this,
				urls='${ctx}/cell/quicksettings/syncANRCellInfos.action',
				params = {
					smallCellCode:'${smallCellCode}'
				},
				str = Math.random().toString();
				
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){
					vm.$message({
						message: '<%=rb.getString("MingLingYiXiaFa")%>',
						type:'success',
					});
					vm.$refs.adjacentRegionTable.refresh();
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
        activeMultFmt(row, value, index) {
            var html = '',
                states = value ? (value+'').split(',') : [];

            if(states && states.length>1) {
                states.map(function(state, idx){
                    if(state == '1') {
                        html += ['<div class="activeStatusItem">',
                                    '<span class="el-icon el-icon-status-active"></span>',
                                    '<span class="status-tip">Cell'+(idx+1)+' Status: <%= rb.getString("JiHuo")%></span>',
                                '</div>'].join(' ');
                    }else {
                        html += ['<div class="inactiveStatusItem">',
                                    '<span class="el-icon el-icon-status-active"></span>',
                                    '<span class="status-tip">Cell'+(idx+1)+' Status: <%= rb.getString("QuJiHuo")%></span>',
                                '</div>'].join(' ');
                    }
                });
            }else {
                if(states[0] == '1') {
                    html += ['<div class="activeStatusItem">',
                                    '<span class="el-icon el-icon-status-active"></span>',
                                    '<span class="status-tip">Cell1 Status: <%= rb.getString("JiHuo")%></span>',
                                '</div>',
                                '<div class="activeStatusItem">',
                                    '<span class="el-icon el-icon-status-active"></span>',
                                    '<span class="status-tip">Cell2 Status: <%= rb.getString("JiHuo")%></span>',
                                '</div>'].join(' ');
                }else {
                    html += ['<div class="inactiveStatusItem">',
                                '<span class="el-icon el-icon-status-active"></span>',
                                '<span class="status-tip">Cell1 Status: <%= rb.getString("QuJiHuo")%></span>',
                            '</div>',
                            '<div class="inactiveStatusItem">',
                                '<span class="el-icon el-icon-status-active"></span>',
                                '<span class="status-tip">Cell2 Status: <%= rb.getString("QuJiHuo")%></span>',
                            '</div>'].join(' ');
                }
            }

            return html;
        },
	},
	mounted(){
		var vm = this;
		eventBus.$off("enb-data").$on("enb-data",this.init)
	}
})
</script> 
