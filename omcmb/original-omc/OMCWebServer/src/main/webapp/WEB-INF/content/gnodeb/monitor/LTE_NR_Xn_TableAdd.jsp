<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#add_LTE_NR_XnTablePage{
		background-color: #FFFFFF;
	}
	#add_LTE_NR_XnTablePage .logHeader{
		height: 50px;
	}
	#add_LTE_NR_XnTablePage .headTitleBox{
		height: 50px;
		width: 100%;
		position: relative;
		border-bottom: 1px solid #E9E9E9;
		display: flex;
		align-items: center;
		font-size: 14px;
		font-weight: bold;
		padding-left: 20px;
	}
	#add_LTE_NR_XnTablePage .logContent{
		height: calc(100% - 80px) !important;
		padding: 30px 0px 30px 20px;
		overflow: auto;
		
	}
	#add_LTE_NR_XnTablePage .el-collapse-item__header{
		border-bottom:1px solid #fff;
	}
	#add_LTE_NR_XnTablePage .el-collapse-item__arrow{
		position:absolute;
		left:20px;
		top:0px;
	}
	#add_LTE_NR_XnTablePage .el-collapse-item{
		position:relative;
	}
	#add_LTE_NR_XnTablePage .el-icon-arrow-right{
		font-size:16px;
	}
	#add_LTE_NR_XnTablePage .el-icon-arrow-right:before{
		content:"\e639";
		color:#BBB;
	}
	#add_LTE_NR_XnTablePage .is-active.el-icon-arrow-right:before{
		content:"\e638";
		color:#BBB;
	}
	#add_LTE_NR_XnTablePage .el-collapse-item__arrow.is-active{
		transform:rotate(0deg);
	}
	#add_LTE_NR_XnTablePage .el-collapse{
		border-top:1px solid #fff;
		border-bottom:1px solid #fff;
		margin-bottom: 80px;
	}
	#add_LTE_NR_XnTablePage .el-collapse-item__wrap{
		border-bottom:1px solid #fff;
	}
	#add_LTE_NR_XnTablePage .el-collapse-item__header{
		max-width:540px;
	}
	#add_LTE_NR_XnTablePage .el-collapse-item__header .el-icon::before{
		font-size: 20px;
	}
	#add_LTE_NR_XnTablePage .el-collapse-item__content{
		padding-bottom: 0px;
	}
	#add_LTE_NR_XnTablePage .el-form-item {
		margin-bottom: 20px;
	}
	#add_LTE_NR_XnTablePage .selectInputCls .el-form-item .el-select>.el-input{
		width: 100px;
	}
	#add_LTE_NR_XnTablePage .el-form-item__error{
		padding-top: 0px;
	}
	#add_LTE_NR_XnTablePage .validate-item .el-input__inner{
		width:190px;
	}
	#add_LTE_NR_XnTablePage .selectErrcCls .el-form-item__error{
		position: absolute;
		left: 120px;
		top:10px !important;
	}
</style>
<div class="panelDefault" id="add_LTE_NR_XnTablePage" style="overflow:hidden">
	<div class="logHeader">
		<div class="headTitleBox">
			{{headTitle}}
			<div style="position:absolute;right:30px;">
				<span class="el-icon el-icon-circle-close" style="margin-left:10px;" @click="closeClick"></span>
			</div>
		</div>
	</div>
	<div class="logContent">
		<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="left" :hide-required-asterisk='true'>
			<el-collapse v-model="nrCellCollapse">
				<el-collapse-item name="LF" v-show="addTableType == 'LF'">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">Freq Setting</span>
						</p>
					</template>
					<div class="rightContentCls" >
						<div style="display:flex;flex-wrap: wrap" class="selectInputCls">
							<el-form-item prop='LF_CarrierFreq' style="width:45%;min-width:540px;" label="CarrierFreq" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LF_CarrierFreq' style="width:150px;padding-top:5px;" maxlength="11">
									<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LF_AllowdMeasBandWidth' style="width:45%;min-width:540px;" label="AllowMeasBandWidth" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.LF_AllowdMeasBandWidth' style="width:60px;padding-top:5px;">
									<el-option label='mbw6' value='mbw6'></el-option>
									<el-option label='mbw15' value='mbw15'></el-option>
									<el-option label="mbw25" value="mbw25"></el-option>
									<el-option label="mbw50" value="mbw50"></el-option>
									<el-option label="mbw75" value="mbw75"></el-option>
									<el-option label="mbw100" value="mbw100"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='LF_PresAntennaPort1' style="width:45%;min-width:540px;" label="PresAntennaPort1" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.LF_PresAntennaPort1' style="width:60px;padding-top:5px;">
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='LF_QOffset' style="width:45%;min-width:540px;" label="QOffset" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.LF_QOffset' style="width:60px;padding-top:5px;">
									<el-option v-for="item in QOffsetDataList" :label='item' :value='item'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='LF_WideBandRsrqMeas' style="width:45%;min-width:540px;" label="WideBandRsrqMeas" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.LF_WideBandRsrqMeas' style="width:60px;padding-top:5px;">
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='LF_CellReselectionPriority' style="width:45%;min-width:540px;" label="CellReselectionPriority" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LF_CellReselectionPriority' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~7,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LF_ThreshXHigh' style="width:45%;min-width:540px;" label="ThreshXHigh" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LF_ThreshXHigh' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LF_ThreshXLow' style="width:45%;min-width:540px;" label="ThreshXLow" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LF_ThreshXLow' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LF_QRxLevMin' style="width:45%;min-width:540px;" label="QRxLevMin" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LF_QRxLevMin' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：-70~-22,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LF_QQualMin' style="width:45%;min-width:540px;" label="QQualMin" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LF_QQualMin' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：-34~-3,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LF_PMaxEUTRA' style="width:45%;min-width:540px;" label="PMaxEUTRA" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LF_PMaxEUTRA' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：-30~33,Integer</template>
								</el-input>
							</el-form-item>
						</div>
					</div>
				</el-collapse-item>
				<el-collapse-item name="LC" v-show="addTableType == 'LC'">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">Cell Neigh Setting</span>
						</p>
					</template>
					<div class="rightContentCls" >
						<div style="display:flex;flex-wrap: wrap" class="selectInputCls">
							<el-form-item prop='LC_PLMNID' style="width:45%;min-width:540px;" label="PLMNID" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LC_PLMNID' style="width:150px;padding-top:5px;">
									<template slot="append">Length：5~6 Digit,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LC_CID' style="width:45%;min-width:540px;" label="CID" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LC_CID' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：1~268435455,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LC_EUTRACarrierARFCN' style="width:45%;min-width:540px;" label="EUTRACarrierARFCN" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LC_EUTRACarrierARFCN' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LC_PhyCellID' style="width:45%;min-width:540px;" label="PhyCellID" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LC_PhyCellID' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~503,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LC_QOffset' style="width:45%;min-width:540px;" label="QOffset" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.LC_QOffset' style="width:60px;padding-top:5px;">
									<el-option v-for="item in QOffsetDataList" :label='item' :value='item'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='LC_QRxLevMinOffsetCell' style="width:45%;min-width:540px;" label="QRxLevMinOffsetCell" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.LC_QRxLevMinOffsetCell' style="width:60px;padding-top:5px;">
									<el-option label='1' value='1'></el-option>
									<el-option label="2" value="2"></el-option>
									<el-option label="3" value="3"></el-option>
									<el-option label='4' value='4'></el-option>
									<el-option label="5" value="5"></el-option>
									<el-option label="6" value="6"></el-option>
									<el-option label='7' value='7'></el-option>
									<el-option label="8" value="8"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='LC_QQualMinOffsetCell' style="width:45%;min-width:540px;" label="QQualMinOffsetCell" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.LC_QQualMinOffsetCell' style="width:60px;padding-top:5px;">
									<el-option label='1' value='1'></el-option>
									<el-option label="2" value="2"></el-option>
									<el-option label="3" value="3"></el-option>
									<el-option label='4' value='4'></el-option>
									<el-option label="5" value="5"></el-option>
									<el-option label="6" value="6"></el-option>
									<el-option label='7' value='7'></el-option>
									<el-option label="8" value="8"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='LC_CIO' style="width:45%;min-width:540px;" label="CIO" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LC_CIO' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：-24~24,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LC_Blacklisted' style="width:45%;min-width:540px;" label="Blacklisted" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.LC_Blacklisted' style="width:60px;padding-top:5px;">
									<el-option label='false' value='0'></el-option>
									<el-option label='true' value='1'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='LC_TAC' style="width:45%;min-width:540px;" label="TAC" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LC_TAC' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LC_eNBType' style="width:45%;min-width:540px;" label="eNB Type" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.LC_eNBType' style="width:60px;padding-top:5px;">
									<el-option label='Macro' value='0'></el-option>
									<el-option label='Home' value='1'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='LC_ECGI' style="width:45%;min-width:540px;" label="ECGI" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.LC_ECGI' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~1048575,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='LC_noRemove' style="width:45%;min-width:540px;" label="No Remove" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.LC_noRemove' style="width:60px;padding-top:5px;">
									<el-option label='false' value='0'></el-option>
									<el-option label='true' value='1'></el-option>
								</el-select>
							</el-form-item>
						</div>
					</div>
				</el-collapse-item>
				<el-collapse-item name="NF" v-show="addTableType == 'NF'">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">InterFREQ Measurement Setting</span>
						</p>
					</template>
					<div class="rightContentCls" >
						<div style="display:flex;flex-wrap: wrap" class="selectInputCls">
							<el-form-item prop='NF_Enable' style="width:45%;min-width:540px;" label="Enable" label-width="190px">
								<el-switch v-model="ruleForm.NF_Enable" active-value="true" inactive-value="false" style='padding-top:10px;'></el-switch>
							</el-form-item>
							<el-form-item prop='NF_SSBFrequency' style="width:45%;min-width:540px;" label="SSBFrequency" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_SSBFrequency' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_SubCarrierSpacing' style="width:45%;min-width:540px;" label="SubCarrierSpacing" label-width="190px" class="selectErrcCls">
								<el-select v-model='ruleForm.NF_SubCarrierSpacing' style="width:60px;padding-top:5px;">
									<el-option label='khz15' value='0'></el-option>
									<el-option label="khz30" value="1"></el-option>
									<el-option label="khz60" value="2"></el-option>
									<el-option label='khz120' value='3'></el-option>
									<el-option label='khz240' value='4'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NF_SmtcPeriodicity' style="width:45%;min-width:540px;" label="SmtcPeriodicity" label-width="190px" class="selectErrcCls">
								<el-select v-model='ruleForm.NF_SmtcPeriodicity' style="width:60px;padding-top:5px;">
									<el-option label='sf5' value='0'></el-option>
									<el-option label='sf10' value='1'></el-option>
									<el-option label="sf20" value="2"></el-option>
									<el-option label="sf40" value="3"></el-option>
									<el-option label="sf80" value="4"></el-option>
									<el-option label="sf160" value="5"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NF_SmtcOffset' style="width:45%;min-width:540px;" label="SmtcOffset" label-width="190px" class='validate-item' >
								<el-input v-model.trim='ruleForm.NF_SmtcOffset' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~159,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_SmtcDuration' style="width:45%;min-width:540px;" label="SmtcDuration" label-width="190px" class="selectErrcCls">
								<el-select v-model='ruleForm.NF_SmtcDuration' style="width:60px;padding-top:5px;">
									<el-option label='sf1' value='0'></el-option>
									<el-option label='sf2' value='1'></el-option>
									<el-option label="sf3" value="2"></el-option>
									<el-option label="sf4" value="3"></el-option>
									<el-option label="sf5" value="4"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NF_SSBlocksConsolidationRsrp' style="width:45%;min-width:540px;" label="SSBlocksConsolidationRsrp" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_SSBlocksConsolidationRsrp' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_SSBlocksConsolidationRsrq' style="width:45%;min-width:540px;" label="SSBlocksConsolidationRsrq" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_SSBlocksConsolidationRsrq' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_SSBlocksConsolidationSinr' style="width:45%;min-width:540px;" label="SSBlocksConsolidationSinr" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_SSBlocksConsolidationSinr' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_NrofSSBlocksToAverage' style="width:45%;min-width:540px;" label="NrofSSBlocksToAverage" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_NrofSSBlocksToAverage' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：2~16,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_RsrpOffsetSSB' style="width:45%;min-width:540px;" label="RsrpOffsetSSB" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_RsrpOffsetSSB' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_RsrqOffsetSSB' style="width:45%;min-width:540px;" label="RsrqOffsetSSB" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_RsrqOffsetSSB' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_SinrOffsetSSB' style="width:45%;min-width:540px;" label="SinrOffsetSSB" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_SinrOffsetSSB' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_RsrpOffsetCsiRs' style="width:45%;min-width:540px;" label="RsrpOffsetCsiRs" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_RsrpOffsetCsiRs' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_RsrqOffsetCsiRs' style="width:45%;min-width:540px;" label="RsrqOffsetCsiRs" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_RsrqOffsetCsiRs' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_SinrOffsetCsiRs' style="width:45%;min-width:540px;" label="SinrOffsetCsiRs" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_SinrOffsetCsiRs' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_BitmapType' style="width:45%;min-width:540px;" label="BitmapType" label-width="190px" class="selectErrcCls">
								<el-select v-model='ruleForm.NF_BitmapType' style="width:60px;padding-top:5px;">
									<el-option label='short' value='0'></el-option>
									<el-option label='medium' value='1'></el-option>
									<el-option label="long" value="2"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NF_Bitmap' style="width:45%;min-width:540px;" label="Bitmap" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_Bitmap' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~18446744073709551615,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_DeriveSSBIndexFromCell' style="width:45%;min-width:540px;" label="DeriveSSBIndexFromCell" label-width="190px" class="selectErrcCls">
								<el-select v-model='ruleForm.NF_DeriveSSBIndexFromCell' style="width:60px;padding-top:5px;">
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NF_FreqBandIndicatorNR' style="width:45%;min-width:540px;" label="FreqBandIndicatorNR" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_FreqBandIndicatorNR' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：1~1024,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_OffsetToPointA' style="width:45%;min-width:540px;" label="Offset To Point A" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_OffsetToPointA' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~2199,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NF_SSBSubCarrierOffset' style="width:45%;min-width:540px;" label="SSB Sub Carrier Offset" label-width="190px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NF_SSBSubCarrierOffset' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
								</el-input>
							</el-form-item>
						</div>
					</div>
				</el-collapse-item>
				<el-collapse-item name="NC" v-show="addTableType == 'NC'">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">Cell Neigh Setting</span>
						</p>
					</template>
					<div class="rightContentCls" >
						<div style="display:flex;flex-wrap: wrap" class="selectInputCls">
							<el-form-item prop='NC_PLMNID' style="width:45%;min-width:540px;" label="PLMNID" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NC_PLMNID' style="width:150px;padding-top:5px;">
									<template slot="append">Length：5~6 Digit,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NC_CID' style="width:45%;min-width:540px;" label="CID" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NC_CID' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~68719476735,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NC_NRARFCN' style="width:45%;min-width:540px;" label="NRCarrierARFCN" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NC_NRARFCN' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NC_SSBFrequency' style="width:45%;min-width:540px;" label="ssbFrequency" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NC_SSBFrequency' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NC_SSBSubcarrierSpacing' style="width:45%;min-width:540px;" label="ssbSubcarrierSpacing" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.NC_SSBSubcarrierSpacing' style="width:60px;padding-top:5px;">
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
									<el-option label="2" value="2"></el-option>
									<el-option label="3" value="3"></el-option>
									<el-option label="4" value="4"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NC_PhyCellID' style="width:45%;min-width:540px;" label="PhyCellID" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NC_PhyCellID' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~1007,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NC_QOffset' style="width:45%;min-width:540px;" label="QOffset" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.NC_QOffset' style="width:60px;padding-top:5px;">
										<el-option v-for="item in QOffsetDataList" :label='item' :value='item'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NC_QRxLevMinOffsetCell' style="width:45%;min-width:540px;" label="QRxLevMinOffsetCell" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.NC_QRxLevMinOffsetCell' style="width:60px;padding-top:5px;">
									<el-option label='1' value='1'></el-option>
									<el-option label="2" value="2"></el-option>
									<el-option label="3" value="3"></el-option>
									<el-option label='4' value='4'></el-option>
									<el-option label="5" value="5"></el-option>
									<el-option label="6" value="6"></el-option>
									<el-option label='7' value='7'></el-option>
									<el-option label="8" value="8"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NC_QQualMinOffsetCell' style="width:45%;min-width:540px;" label="QQualMinOffsetCell" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.NC_QQualMinOffsetCell' style="width:60px;padding-top:5px;">
									<el-option label='1' value='1'></el-option>
									<el-option label="2" value="2"></el-option>
									<el-option label="3" value="3"></el-option>
									<el-option label='4' value='4'></el-option>
									<el-option label="5" value="5"></el-option>
									<el-option label="6" value="6"></el-option>
									<el-option label='7' value='7'></el-option>
									<el-option label="8" value="8"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NC_CIO' style="width:45%;min-width:540px;" label="CIO" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.NC_CIO' style="width:60px;padding-top:5px;">
										<el-option v-for="item in CIODataList" :label='item' :value='item'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NC_Blacklisted' style="width:45%;min-width:540px;" label="Blacklisted" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.NC_Blacklisted' style="width:60px;padding-top:5px;">
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NC_TAC' style="width:45%;min-width:540px;" label="TAC" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NC_TAC' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~16777215,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NC_noRemove' style="width:45%;min-width:540px;" label="No Remove" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.NC_noRemove' style="width:60px;padding-top:5px;">
									<el-option label='false' value='0'></el-option>
									<el-option label='true' value='1'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='NC_gnbIdLength' style="width:45%;min-width:540px;" label="gNB ID Length" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.NC_gnbIdLength' style="width:150px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：22~32,Integer</template>
								</el-input>
							</el-form-item>
						</div>
					</div>
				</el-collapse-item>
				<el-collapse-item name="Xn" v-show="addTableType == 'Xn'">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">Xn Setting</span>
						</p>
					</template>
					<div class="rightContentCls" >
						<div style="display:flex;flex-wrap: wrap" class="selectInputCls">
							<el-form-item prop='Xn_PLMNID' style="width:45%;min-width:540px;" label="PLMNID" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.Xn_PLMNID' style="width:150px;padding-top:5px;">
									<template slot="append">Length：5~6 Digit,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='Xn_RemoteAddress' style="width:45%;min-width:540px;" label="RemoteAddress" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.Xn_RemoteAddress' style="width:150px;padding-top:5px;">
									<template slot="append">Example：1.1.1.1</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='Xn_LinkEnable' style="width:45%;min-width:540px;" label="XnLinkEnable" label-width="160px">
								<el-switch v-model="ruleForm.Xn_LinkEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
							</el-form-item>
							<el-form-item prop='Xn_HoEnable' style="width:45%;min-width:540px;" label="XnHoEnable" label-width="160px">
								<el-switch v-model="ruleForm.Xn_HoEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
							</el-form-item>
						</div>
					</div>
				</el-collapse-item>
				<el-collapse-item name="AMF" v-show="addTableType == 'AMF'">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">AMF Setting</span>
						</p>
					</template>
					<div class="rightContentCls" >
						<div style="display:flex;flex-wrap: wrap" class="selectInputCls">
							<el-form-item prop='AMF_IP' style="width:45%;min-width:540px;" label="AMF IP" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.AMF_IP' style="width:150px;padding-top:5px;">
									<template slot="append">Example：1.1.1.1</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='AMF_PLMNID' style="width:45%;min-width:540px;" label="PLMN ID" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.AMF_PLMNID' style="width:150px;padding-top:5px;">
									<template slot="append">Length：5~6 Digit,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='AMF_Default' style="width:45%;min-width:540px;" label="Default" label-width="160px" class="selectErrcCls">
								<el-select v-model='ruleForm.AMF_Default' style="width:60px;padding-top:5px;">
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
								</el-select>
							</el-form-item>
						</div>
					</div>
				</el-collapse-item>
			</el-collapse>
			<div class="footer">
				<div class="lnkbuttonGroup" style="padding:10px 0 0;">
					<el-button type="primary" @click="addTableSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="closeClick"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</el-form>
	</div>
</div>

<script type="text/javascript">
	
	var nrCellVue = new Vue({
		el: '#add_LTE_NR_XnTablePage',
		data(){
			var vm = this;
			var validateRange = (rule,value,callback)=>{
					var min = rule.min;
					var max = rule.max;
					var mag = rule.mag;
					var addTableType = rule.addTableType || '';
					var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
					if(this.addTableType != addTableType){
						callback();
					}else{
						if(value == '' || value == undefined || value == null){
							callback(new Error(mag))
						}else{
							if(reg.test(value) && value >= min && value <= max){
								callback();
							}else{
								callback(new Error(mag))
							}
						}
					}
				},
				validateBitmapRange = (rule,value,callback)=>{
					var min = rule.min;
					var max = rule.max;
					var mag = rule.mag;
					var addTableType = rule.addTableType || '';
					var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
					if(this.addTableType != addTableType){
						callback();
					}else{
						if(value == '' || value == undefined || value == null){
							callback(new Error(mag))
						}else{
							if(reg.test(value) && this.isLessThan(value,max) && this.isLessThan(min,value)){
								callback()
							}else{
								callback(new Error(mag))
							}
						}
					}
				},
				validateRequired = (rule,value,callback)=>{
					var addTableType = rule.addTableType || '';
					var mag = rule.mag;
					if(this.addTableType != addTableType){
						callback();
					}else{
						if(value == '' || value == undefined || value == null){
							callback(new Error(mag))
						}else{
							callback();
						}
					}
				},
				validateLTE_PLMNID = (rule,value,callback) => {
					var reg = /^[0-9]{5,6}$/
					if(vm.addTableType !== 'LC'){
						callback()
					}else{
						if(value === ''){
							callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
						}else{
							if(reg.test(value)){
								callback();
							}else{
								callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
							}
						} 
					}
				},
				
				validateNC_PLMNID = (rule,value,callback) => {
					var reg = /^[0-9]{5,6}$/
					if(vm.addTableType !== 'NC'){
						callback()
					}else{
						if(value === ''){
							callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
						}else{
							 if(reg.test(value)){
								callback();
							}else{
								callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
							}
						}
					}
				},
				validateXn_PLMNID = (rule,value,callback) => {
					var reg = /^[0-9]{5,6}$/
					if(vm.addTableType !== 'Xn'){
						callback()
					}else{
						if(value === ''){
							callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
						}else{
							 if(reg.test(value)){
								callback();
							}else{
								callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
							}
						}
					}
				},
				validateXn_RemoteAddress= (rule,value,callback) => {
					var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
					if(vm.addTableType !== 'Xn'){
						callback()
					}else{
						if(value === ''){
							callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
						}else{
							 if(vm.isValidIP(value) || vm.isIPv6(value)){
								callback();
							}else{
								callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
							}
						}
					}
				},
				validateAMF_IPAddress= (rule,value,callback) => {
					var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
					if(vm.addTableType !== 'AMF'){
						callback()
					}else{
						if(value === ''){
							callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
						}else{
							if(vm.isValidIP(value) || vm.isIPv6(value)){
								callback();
							}else{
								callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
							}
						}
					}
				},
				validateAMF_PLMNID = (rule,value,callback) => {
					var reg = /^[0-9]{5,6}$/
					if(vm.addTableType !== 'AMF'){
						callback()
					}else{
						if(value === ''){
							callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
						}else{
							if(reg.test(value)){
								callback();
							}else{
								callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
							}
						}
					}
				};
				return {
					addTableType:'LC', //新增表格参数类型  LF LC  NF NC Xn
					optType:'add',   // 操作类型  add 新增   edit 修改
					nrCellCollapse:'LC',
					ruleForm:{
						LF_CarrierFreq:'',
						LF_AllowdMeasBandWidth:'',
						LF_PresAntennaPort1:'',
						LF_QOffset:'',
						LF_WideBandRsrqMeas:'',
						LF_CellReselectionPriority:'',
						LF_ThreshXHigh:'',
						LF_ThreshXLow:'',
						LF_QRxLevMin:'',
						LF_QQualMin:'',
						LF_PMaxEUTRA:'',

						LC_PLMNID:'',
						LC_CID:'',
						LC_EUTRACarrierARFCN:'',
						LC_PhyCellID:'',
						LC_QOffset:'',
						LC_QRxLevMinOffsetCell:'',
						LC_QQualMinOffsetCell:'',
						LC_CIO:'',
						LC_Blacklisted:'',
						LC_TAC:'',
						LC_eNBType:'',
						LC_ECGI:'',
						LC_noRemove:'',

						NF_Enable:'false',
						NF_SSBFrequency:'',
						NF_SubCarrierSpacing:'',
						NF_SmtcPeriodicity:'',
						NF_SmtcOffset:'',
						NF_SmtcDuration:'',
						NF_SSBlocksConsolidationRsrp:'',
						NF_SSBlocksConsolidationRsrq:'',
						NF_SSBlocksConsolidationSinr:'',
						NF_NrofSSBlocksToAverage:'',
						NF_RsrpOffsetSSB:'',
						NF_RsrqOffsetSSB:'',
						NF_SinrOffsetSSB:'',
						NF_RsrpOffsetCsiRs:'',
						NF_RsrqOffsetCsiRs:'',
						NF_SinrOffsetCsiRs:'',
						NF_BitmapType:'',
						NF_Bitmap:'',
						NF_DeriveSSBIndexFromCell:'',
						NF_FreqBandIndicatorNR:'',
						NF_OffsetToPointA:'',
						NF_SSBSubCarrierOffset:'',
						
						NC_PLMNID:'',
						NC_CID:'',
						NC_NRARFCN:'',
						NC_SSBFrequency:'',
						NC_SSBSubcarrierSpacing:'',
						NC_PhyCellID:'',
						NC_QOffset:'',
						NC_QRxLevMinOffsetCell:'',
						NC_QQualMinOffsetCell:'',
						NC_CIO:'',
						NC_Blacklisted:'',
						NC_TAC:'',
						NC_noRemove:'',
						NC_gnbIdLength:'',

						Xn_PLMNID:'',
						Xn_RemoteAddress:'',
						Xn_LinkEnable:'0',
						Xn_HoEnable:'0',

						AMF_IP:'',
						AMF_PLMNID:'',
						AMF_Default:'0'
					},
					rules:{
						LF_CarrierFreq:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer',addTableType:'LF'}],
						LF_CellReselectionPriority:[{validator:validateRange,min:0,max:7,mag:'<%=rb.getString("FanWei")%>：0~7,Integer',addTableType:'LF'}],
						LF_ThreshXHigh:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',addTableType:'LF'}],
						LF_ThreshXLow:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',addTableType:'LF'}],
						LF_QRxLevMin:[{validator:validateRange,min:-70,max:-22,mag:'<%=rb.getString("FanWei")%>：-70~-22,Integer',addTableType:'LF'}],
						LF_QQualMin:[{validator:validateRange,min:-34,max:-3,mag:'<%=rb.getString("FanWei")%>：-34~-3,Integer',addTableType:'LF'}],
						LF_PMaxEUTRA:[{validator:validateRange,min:-30,max:33,mag:'<%=rb.getString("FanWei")%>：-30~33,Integer',addTableType:'LF'}],
						LF_AllowdMeasBandWidth:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'LF'}],
						LF_PresAntennaPort1:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'LF'}],
						LF_QOffset:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'LF'}],
						LF_WideBandRsrqMeas:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'LF'}],

						LC_PLMNID:[{validator:validateLTE_PLMNID,trigger:'blur'}],
						LC_CID:[{validator:validateRange,min:1,max:268435455,mag:'<%=rb.getString("FanWei")%>：1~268435455,Integer',addTableType:'LC'}],
						LC_EUTRACarrierARFCN:[{validator:validateRange,min:0,max:65535,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer',addTableType:'LC'}],
						LC_PhyCellID:[{validator:validateRange,min:0,max:503,mag:'<%=rb.getString("FanWei")%>：0~503,Integer',addTableType:'LC'}],
						LC_CIO:[{validator:validateRange,min:-24,max:24,mag:'<%=rb.getString("FanWei")%>：-24~24,Integer',addTableType:'LC'}],
						LC_TAC:[{validator:validateRange,min:0,max:65535,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer',addTableType:'LC'}],
						LC_ECGI:[{validator:validateRange,min:0,max:1048575,mag:'<%=rb.getString("FanWei")%>：0~1048575,Integer',addTableType:'LC'}],
						LC_QRxLevMinOffsetCell:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'LC'}],
						LC_QQualMinOffsetCell:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'LC'}],
						LC_eNBType:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'LC'}],
						LC_Blacklisted:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'LC'}],
						LC_QOffset:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'LC'}],
						LC_noRemove:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'LC'}],

						NF_SSBFrequency:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer',addTableType:'NF'}],
						NF_SmtcOffset:[{validator:validateRange,min:0,max:159,mag:'<%=rb.getString("FanWei")%>：0~159,Integer',addTableType:'NF'}],
						NF_SSBlocksConsolidationRsrp:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',addTableType:'NF'}],
						NF_SSBlocksConsolidationRsrq:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',addTableType:'NF'}],
						NF_SSBlocksConsolidationSinr:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',addTableType:'NF'}],
						NF_NrofSSBlocksToAverage:[{validator:validateRange,min:2,max:16,mag:'<%=rb.getString("FanWei")%>：2~16,Integer',addTableType:'NF'}],
						NF_RsrpOffsetSSB:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',addTableType:'NF'}],
						NF_RsrqOffsetSSB:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',addTableType:'NF'}],
						NF_SinrOffsetSSB:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',addTableType:'NF'}],
						NF_RsrpOffsetCsiRs:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',addTableType:'NF'}],
						NF_RsrqOffsetCsiRs:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',addTableType:'NF'}],
						NF_SinrOffsetCsiRs:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',addTableType:'NF'}],
						NF_Bitmap:[{validator:validateBitmapRange,min:'0',max:'18446744073709551615',mag:'<%=rb.getString("FanWei")%>：0~18446744073709551615,Integer',addTableType:'NF'}],
						NF_FreqBandIndicatorNR:[{validator:validateRange,min:1,max:1024,mag:'<%=rb.getString("FanWei")%>：1~1024,Integer',addTableType:'NF'}],
						NF_OffsetToPointA:[{validator:validateRange,min:0,max:2199,mag:'<%=rb.getString("FanWei")%>：0~2199,Integer',addTableType:'NF'}],
						NF_SSBSubCarrierOffset:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',addTableType:'NF'}],
						NF_SubCarrierSpacing:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NF'}],
						NF_SmtcPeriodicity:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NF'}],
						NF_SmtcDuration:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NF'}],
						NF_BitmapType:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NF'}],
						NF_DeriveSSBIndexFromCell:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NF'}],
						

						NC_PLMNID:[{validator:validateNC_PLMNID,trigger:'blur'}],
						NC_CID:[{validator:validateRange,min:0,max:68719476735,mag:'<%=rb.getString("FanWei")%>：0~68719476735,Integer',addTableType:'NC'}],
						NC_NRARFCN:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer',addTableType:'NC'}],
						NC_SSBFrequency:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer',addTableType:'NC'}],
						NC_PhyCellID:[{validator:validateRange,min:0,max:1007,mag:'<%=rb.getString("FanWei")%>：0~1007,Integer',addTableType:'NC'}],
						
						NC_TAC:[{validator:validateRange,min:0,max:16777215,mag:'<%=rb.getString("FanWei")%>：0~16777215,Integer',addTableType:'NC'}],
						Xn_PLMNID:[{validator:validateXn_PLMNID,trigger:'blur'}],
						Xn_RemoteAddress:[{validator:validateXn_RemoteAddress,trigger:'blur'}],
						NC_CIO:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NC'}],
						NC_SSBSubcarrierSpacing:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NC'}],
						NC_QOffset:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NC'}],
						NC_QRxLevMinOffsetCell:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NC'}],
						NC_QQualMinOffsetCell:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NC'}],
						NC_Blacklisted:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NC'}],
						NC_noRemove:[{validator:validateRequired,mag:'required',trigger:'change',addTableType:'NC'}],
						NC_gnbIdLength:[{validator:validateRange,min:22,max:32,mag:'<%=rb.getString("FanWei")%>：22~32,Integer',addTableType:'NC'}],
						
						AMF_IP:[{validator:validateAMF_IPAddress,trigger:'blur'}],
						AMF_PLMNID:[{validator:validateAMF_PLMNID,trigger:'blur'}],
						
					},
					errorMessage:'',
					QOffsetDataList:['-24','-22','-20','-18','-16','-14','-12','-10','-8','-6','-5','-4','-3','-2','-1',
					'0','1','2','3','4','5','6','8','10','12','14','16','18','20','22','24'],
					CIODataList:['-24','-22','-20','-18','-16','-14','-12','-10','-8','-6','-5','-4','-3','-2','-1',
					'0','1','2','3','4','5','6','8','10','12','14','16','18','20','22','24'],
				}
			},
		computed: {
			headTitle() {
				var codeList={
						'LF':'LTE N-FREQ',
						'LC':'LTE N-CELL',
						'NF':'NR N-FREQ',
						'NC':'NR N-CELL',
						'Xn':'Xn',
						'AMF':'AMF'
					},
					optType={
						'add':'Add',
						'edit':'Modify'
					};

				return optType[this.optType]+' '+ codeList[this.addTableType]
			},
			cellIndex(){
				var cellName = gnbQuickSettingPageVue.activeName,
					codes={
						'cell1':'1',
						'cell2':'2',
						'cell3':'3',
						'cell4':'4',
					};
				return codes[cellName]
			}
		},
		watch: {
			
		},
		methods: {
			// 初始化
			init(row,optType,addTableType){
				var vm = this;
				vm.optType = optType;
				vm.addTableType = addTableType;
				vm.nrCellCollapse = addTableType;
				Object.assign(vm.ruleForm, row);
				initForm(vm.$refs.ruleForm);
			},
			
			// 验证输入的是否是整数
			isInteger(str) {
				if(str.length==0){
					return false;
				}
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
				if(!reg.test(str)){
					return false;
				}
				return true;  
			},
			addTableSubmit(){
				var vm = this,
					params={};
				
				if(vm.addTableType == 'LF'){
					Object.keys(vm.ruleForm).forEach(function(key){
						if(key.substring(0,2) == 'LF'){
							params[key] = vm.ruleForm[key]
						}
					})
					if(vm.ruleForm.operateType){
						params.operateType = vm.ruleForm.operateType
					}
					vm.$refs.ruleForm.validate(function(valid){
						if(valid){
							if(vm.cellIndex == '1'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell1InfoVue.ruleForm.LF_List.length == 0){
										params.LF_idx = '1'
									}else{
										var idList=[];
										cell1InfoVue.ruleForm.LF_List.map((item)=>{
											idList.push(item.LF_idx);
										})
										params.LF_idx = vm.createId(1,idList); 
									}
									
									cell1InfoVue.ruleForm.LF_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.LF_idx = vm.ruleForm.LF_idx;
									var idx='';
									cell1InfoVue.ruleForm.LF_List.map((item,index)=>{
										if(item.LF_idx == params.LF_idx){
											idx = index
										}
									})
									Object.assign(cell1InfoVue.ruleForm.LF_List[idx],params)
								}
							}else if(vm.cellIndex == '2'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell2InfoVue.ruleForm.LF_List.length == 0){
										params.LF_idx = '1'
									}else{
										var idList=[];
										cell2InfoVue.ruleForm.LF_List.map((item)=>{
											idList.push(item.LF_idx);
										})
										params.LF_idx = vm.createId(1,idList); 
									}
									cell2InfoVue.ruleForm.LF_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.LF_idx = vm.ruleForm.LF_idx;
									var idx='';
									cell2InfoVue.ruleForm.LF_List.map((item,index)=>{
										if(item.LF_idx == params.LF_idx){
											idx = index
										}
									})
									Object.assign(cell2InfoVue.ruleForm.LF_List[idx],params)
								}
							}else if(vm.cellIndex == '3'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell3InfoVue.ruleForm.LF_List.length == 0){
										params.LF_idx = '1'
									}else{
										var idList=[];
										cell3InfoVue.ruleForm.LF_List.map((item)=>{
											idList.push(item.LF_idx);
										})
										params.LF_idx = vm.createId(1,idList); 
									}
									cell3InfoVue.ruleForm.LF_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.LF_idx = vm.ruleForm.LF_idx;
									var idx='';
									cell3InfoVue.ruleForm.LF_List.map((item,index)=>{
										if(item.LF_idx == params.LF_idx){
											idx = index
										}
									})
									Object.assign(cell3InfoVue.ruleForm.LF_List[idx],params)
								}
							}else if(vm.cellIndex == '4'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell4InfoVue.ruleForm.LF_List.length == 0){
										params.LF_idx = '1'
									}else{
										var idList=[];
										cell4InfoVue.ruleForm.LF_List.map((item)=>{
											idList.push(item.LF_idx);
										})
										params.LF_idx = vm.createId(1,idList); 
									}
									cell4InfoVue.ruleForm.LF_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.LF_idx = vm.ruleForm.LF_idx;
									var idx='';
									cell4InfoVue.ruleForm.LF_List.map((item,index)=>{
										if(item.LF_idx == params.LF_idx){
											idx = index
										}
									})
									Object.assign(cell4InfoVue.ruleForm.LF_List[idx],params)
								}
							}
							//eventBus.$emit('close-sharingSlide');
							gnbQuickSettingPageVue.$refs.sharingSlide.hide();
						}
					})
				}else if(vm.addTableType == 'LC'){
					Object.keys(vm.ruleForm).forEach(function(key){
						if(key.substring(0,2) == 'LC'){
							params[key] = vm.ruleForm[key]
						}
					})
					if(vm.ruleForm.operateType){
						params.operateType = vm.ruleForm.operateType
					}
					vm.$refs.ruleForm.validate(function(valid){
						if(valid){
							if(vm.cellIndex == '1'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell1InfoVue.ruleForm.LC_List.length == 0){
										params.LC_idx = '1'
									}else{
										var idList=[];
										cell1InfoVue.ruleForm.LC_List.map((item)=>{
											idList.push(item.LC_idx);
										})
										params.LC_idx = vm.createId(1,idList); 
									}
									
									cell1InfoVue.ruleForm.LC_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.LC_idx = vm.ruleForm.LC_idx;
									var idx='';
									cell1InfoVue.ruleForm.LC_List.map((item,index)=>{
										if(item.LC_idx == params.LC_idx){
											idx = index
										}
									})
									Object.assign(cell1InfoVue.ruleForm.LC_List[idx],params)
								}
							}else if(vm.cellIndex == '2'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell2InfoVue.ruleForm.LC_List.length == 0){
										params.LC_idx = '1'
									}else{
										var idList=[];
										cell2InfoVue.ruleForm.LC_List.map((item)=>{
											idList.push(item.LC_idx);
										})
										params.LC_idx = vm.createId(1,idList); 
									}
									cell2InfoVue.ruleForm.LC_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.LC_idx = vm.ruleForm.LC_idx;
									var idx='';
									cell2InfoVue.ruleForm.LC_List.map((item,index)=>{
										if(item.LC_idx == params.LC_idx){
											idx = index
										}
									})
									Object.assign(cell2InfoVue.ruleForm.LC_List[idx],params)
								}
							}else if(vm.cellIndex == '3'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell3InfoVue.ruleForm.LC_List.length == 0){
										params.LC_idx = '1'
									}else{
										var idList=[];
										cell3InfoVue.ruleForm.LC_List.map((item)=>{
											idList.push(item.LC_idx);
										})
										params.LC_idx = vm.createId(1,idList); 
									}
									cell3InfoVue.ruleForm.LC_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.LC_idx = vm.ruleForm.LC_idx;
									var idx='';
									cell3InfoVue.ruleForm.LC_List.map((item,index)=>{
										if(item.LC_idx == params.LC_idx){
											idx = index
										}
									})
									Object.assign(cell3InfoVue.ruleForm.LC_List[idx],params)
								}
							}else if(vm.cellIndex == '4'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell4InfoVue.ruleForm.LC_List.length == 0){
										params.LC_idx = '1'
									}else{
										var idList=[];
										cell4InfoVue.ruleForm.LC_List.map((item)=>{
											idList.push(item.LC_idx);
										})
										params.LC_idx = vm.createId(1,idList); 
									}
									cell4InfoVue.ruleForm.LC_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.LC_idx = vm.ruleForm.LC_idx;
									var idx='';
									cell4InfoVue.ruleForm.LC_List.map((item,index)=>{
										if(item.LC_idx == params.LC_idx){
											idx = index
										}
									})
									Object.assign(cell4InfoVue.ruleForm.LC_List[idx],params)
								}
							}
							//eventBus.$emit('close-sharingSlide');
							gnbQuickSettingPageVue.$refs.sharingSlide.hide();
						}
					})
				}else if(vm.addTableType == 'NF'){
					Object.keys(vm.ruleForm).forEach(function(key){
						if(key.substring(0,2) == 'NF'){
							params[key] = vm.ruleForm[key]
						}
					})
					if(vm.ruleForm.operateType){
						params.operateType = vm.ruleForm.operateType
					}
					vm.$refs.ruleForm.validate(function(valid){
						if(valid){
							if(vm.cellIndex == '1'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell1InfoVue.ruleForm.NF_List.length == 0){
										params.NF_idx = '1'
									}else{
										var idList=[];
										cell1InfoVue.ruleForm.NF_List.map((item)=>{
											idList.push(item.NF_idx);
										})
										params.NF_idx = vm.createId(1,idList); 
									}
									
									cell1InfoVue.ruleForm.NF_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.NF_idx = vm.ruleForm.NF_idx;
									var idx='';
									cell1InfoVue.ruleForm.NF_List.map((item,index)=>{
										if(item.NF_idx == params.NF_idx){
											idx = index
										}
									})
									Object.assign(cell1InfoVue.ruleForm.NF_List[idx],params)
								}
							}else if(vm.cellIndex == '2'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell2InfoVue.ruleForm.NF_List.length == 0){
										params.NF_idx = '1'
									}else{
										var idList=[];
										cell2InfoVue.ruleForm.NF_List.map((item)=>{
											idList.push(item.NF_idx);
										})
										params.NF_idx = vm.createId(1,idList); 
									}
									cell2InfoVue.ruleForm.NF_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.NF_idx = vm.ruleForm.NF_idx;
									var idx='';
									cell2InfoVue.ruleForm.NF_List.map((item,index)=>{
										if(item.NF_idx == params.NF_idx){
											idx = index
										}
									})
									Object.assign(cell2InfoVue.ruleForm.NF_List[idx],params)
								}
							}else if(vm.cellIndex == '3'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell3InfoVue.ruleForm.NF_List.length == 0){
										params.NF_idx = '1'
									}else{
										var idList=[];
										cell3InfoVue.ruleForm.NF_List.map((item)=>{
											idList.push(item.NF_idx);
										})
										params.NF_idx = vm.createId(1,idList); 
									}
									cell3InfoVue.ruleForm.NF_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.NF_idx = vm.ruleForm.NF_idx;
									var idx='';
									cell3InfoVue.ruleForm.NF_List.map((item,index)=>{
										if(item.NF_idx == params.NF_idx){
											idx = index
										}
									})
									Object.assign(cell3InfoVue.ruleForm.NF_List[idx],params)
								}
							}else if(vm.cellIndex == '4'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell4InfoVue.ruleForm.NF_List.length == 0){
										params.NF_idx = '1'
									}else{
										var idList=[];
										cell4InfoVue.ruleForm.NF_List.map((item)=>{
											idList.push(item.NF_idx);
										})
										params.NF_idx = vm.createId(1,idList); 
									}
									cell4InfoVue.ruleForm.NF_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.NF_idx = vm.ruleForm.NF_idx;
									var idx='';
									cell4InfoVue.ruleForm.NF_List.map((item,index)=>{
										if(item.NF_idx == params.NF_idx){
											idx = index
										}
									})
									Object.assign(cell4InfoVue.ruleForm.NF_List[idx],params)
								}
							}
							//eventBus.$emit('close-sharingSlide');
							gnbQuickSettingPageVue.$refs.sharingSlide.hide();
						}
					})
				}else if(vm.addTableType == 'NC'){
					Object.keys(vm.ruleForm).forEach(function(key){
						if(key.substring(0,2) == 'NC'){
							params[key] = vm.ruleForm[key]
						}
					})
					if(vm.ruleForm.operateType){
						params.operateType = vm.ruleForm.operateType
					}
					vm.$refs.ruleForm.validate(function(valid){
						if(valid){
							if(vm.cellIndex == '1'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell1InfoVue.ruleForm.NC_List.length == 0){
										params.NC_idx = '1'
									}else{
										var idList=[];
										cell1InfoVue.ruleForm.NC_List.map((item)=>{
											idList.push(item.NC_idx);
										})
										params.NC_idx = vm.createId(1,idList); 
									}
									
									cell1InfoVue.ruleForm.NC_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.NC_idx = vm.ruleForm.NC_idx;
									var idx='';
									cell1InfoVue.ruleForm.NC_List.map((item,index)=>{
										if(item.NC_idx == params.NC_idx){
											idx = index
										}
									})
									Object.assign(cell1InfoVue.ruleForm.NC_List[idx],params)
								}
							}else if(vm.cellIndex == '2'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell2InfoVue.ruleForm.NC_List.length == 0){
										params.NC_idx = '1'
									}else{
										var idList=[];
										cell2InfoVue.ruleForm.NC_List.map((item)=>{
											idList.push(item.NC_idx);
										})
										params.NC_idx = vm.createId(1,idList); 
									}
									cell2InfoVue.ruleForm.NC_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.NC_idx = vm.ruleForm.NC_idx;
									var idx='';
									cell2InfoVue.ruleForm.NC_List.map((item,index)=>{
										if(item.NC_idx == params.NC_idx){
											idx = index
										}
									})
									Object.assign(cell2InfoVue.ruleForm.NC_List[idx],params)
								}
							}else if(vm.cellIndex == '3'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell3InfoVue.ruleForm.NC_List.length == 0){
										params.NC_idx = '1'
									}else{
										var idList=[];
										cell3InfoVue.ruleForm.NC_List.map((item)=>{
											idList.push(item.NC_idx);
										})
										params.NC_idx = vm.createId(1,idList); 
									}
									cell3InfoVue.ruleForm.NC_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.NC_idx = vm.ruleForm.NC_idx;
									var idx='';
									cell3InfoVue.ruleForm.NC_List.map((item,index)=>{
										if(item.NC_idx == params.NC_idx){
											idx = index
										}
									})
									Object.assign(cell3InfoVue.ruleForm.NC_List[idx],params)
								}
							}else if(vm.cellIndex == '4'){
								if(vm.optType == 'add'){
									params.operateType = 'add'
									if(cell4InfoVue.ruleForm.NC_List.length == 0){
										params.NC_idx = '1'
									}else{
										var idList=[];
										cell4InfoVue.ruleForm.NC_List.map((item)=>{
											idList.push(item.NC_idx);
										})
										params.NC_idx = vm.createId(1,idList); 
									}
									cell4InfoVue.ruleForm.NC_List.push(params);
								}else{
									if(params.operateType && params.operateType == 'add'){
										params.operateType = 'add'
									}else{
										params.operateType = 'edit';
									}
									params.NC_idx = vm.ruleForm.NC_idx;
									var idx='';
									cell4InfoVue.ruleForm.NC_List.map((item,index)=>{
										if(item.NC_idx == params.NC_idx){
											idx = index
										}
									})
									Object.assign(cell4InfoVue.ruleForm.NC_List[idx],params)
								}
							}
							//eventBus.$emit('close-sharingSlide');
							gnbQuickSettingPageVue.$refs.sharingSlide.hide();
						}
					})
				}else if(vm.addTableType == 'Xn'){
					Object.keys(vm.ruleForm).forEach(function(key){
						if(key.substring(0,2) == 'Xn'){
							params[key] = vm.ruleForm[key]
						}
					})
					if(vm.ruleForm.operateType){
						params.operateType = vm.ruleForm.operateType
					}
					vm.$refs.ruleForm.validate(function(valid){
						if(valid){
							if(vm.optType == 'add'){
								params.operateType = 'add'
								if(gnbQuickSettingPageVue.ruleForm.XnList.length == 0){
									params.Xn_idx = '1'
								}else{
									var idList=[];
									gnbQuickSettingPageVue.ruleForm.XnList.map((item)=>{
										idList.push(item.Xn_idx);
									})
									params.Xn_idx = vm.createId(1,idList); 
								}
								gnbQuickSettingPageVue.ruleForm.XnList.push(params);
							}else{
								if(params.operateType && params.operateType == 'add'){
									params.operateType = 'add'
								}else{
									params.operateType = 'edit';
								}
								params.Xn_idx = vm.ruleForm.Xn_idx;
								var idx='';
								gnbQuickSettingPageVue.ruleForm.XnList.map((item,index)=>{
									if(item.Xn_idx == params.Xn_idx){
										idx = index
									}
								})
								Object.assign(gnbQuickSettingPageVue.ruleForm.XnList[idx],params)
							}
							//eventBus.$emit('close-sharingSlide');
							gnbQuickSettingPageVue.$refs.sharingSlide.hide();
						}
					})
				}else if(vm.addTableType == 'AMF'){
					Object.keys(vm.ruleForm).forEach(function(key){
						if(key.substring(0,3) == 'AMF'){
							params[key] = vm.ruleForm[key]
						}
					})
					if(vm.ruleForm.operateType){
						params.operateType = vm.ruleForm.operateType
					}
					vm.$refs.ruleForm.validate(function(valid){
						if(valid){
							if(vm.optType == 'add'){
								params.operateType = 'add'
								if(gnbQuickSettingPageVue.ruleForm.AMFList.length == 0){
									params.AMF_idx = '1'
								}else{
									var idList=[];
									gnbQuickSettingPageVue.ruleForm.AMFList.map((item)=>{
										idList.push(item.AMF_idx);
									})
									params.AMF_idx = vm.createId(1,idList); 
								}
								gnbQuickSettingPageVue.ruleForm.AMFList.push(params);
							}else{
								if(params.operateType && params.operateType == 'add'){
									params.operateType = 'add'
								}else{
									params.operateType = 'edit';
								}
								params.AMF_idx = vm.ruleForm.AMF_idx;
								var idx='';
								gnbQuickSettingPageVue.ruleForm.AMFList.map((item,index)=>{
									if(item.AMF_idx == params.AMF_idx){
										idx = index
									}
								})
								Object.assign(gnbQuickSettingPageVue.ruleForm.AMFList[idx],params)
							}
							//eventBus.$emit('close-sharingSlide');
							gnbQuickSettingPageVue.$refs.sharingSlide.hide();
						}
					})
				}
				
			},
			createId(idVal,list){
				var vm = this,
					val = idVal + '';
				if(list.includes(val) == true){
					idVal += 1 ;
					return vm.createId(idVal,list);
				}else{
					return  idVal + '';
				}
			},
			closeClick(){
				var vm = this;

				if(isFormChanged(this.$refs.ruleForm)){
					vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						//eventBus.$emit('close-sharingSlide');
						gnbQuickSettingPageVue.$refs.sharingSlide.hide();
					}).catch(() => {
						
					})
				}else{
					//eventBus.$emit('close-sharingSlide');
					gnbQuickSettingPageVue.$refs.sharingSlide.hide();
				}
			},
			isLessThan(val,target){
				var max = target + '',
				    num = val + '',
					bool = true,
					list = [];
				if(num.length > max.length || isNaN(num)){
					bool = false;
				}
				if(num.length == max.length){
					for(var i = 0;i<max.length;i++){
						var isBig = max[i] - num[i] >= 0;
						list.push(max[i] - num[i] > 0);
						if(!isBig){
							var some = list.filter(function(item){return item == true;});
							if(some.length == 0){
								bool = false;
								break;
							}

						}
					}
				}
				return bool;
			},
			//校验IP
			isValidIP(ip){
				var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
				return reg.test(ip);     
			},
			//Ipv6校验 
			isIPv6(str){ 
				var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
				return reg.test(str);
			},
		},
		created(){},
		mounted(){
			eventBus.$off('addTable-init').$on('addTable-init',this.init);
		}
	});
</script>