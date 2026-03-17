<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<div id='ltePanel' style='height: 100%;overflow: auto;background: #fff;min-width: 900px;'>
	<el-form ref="lteForm" :model="lteForm" :rules="lteRules" 
		label-position="top" style='width:100%;height:100%;' inline>
		<el-collapse v-model="activeNames">
			<el-collapse-item name="neigh">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">LTE Neigh Freq/Cell</span>
					</p>
				</template>
				<div style='margin-top:5px;width:95%;display:flex'>
					<span style='font-size:12px;font-weight:bold;flex:1'>
						Neigh Freq List 
						<span style="padding-left: 5px;font-weight: normal;color: #999;">LTE neighbor frequency can configure up to 8</span> 
					</span>
					<span v-if="lteForm.NeighFreqList.length < 8" @click='addFreq' class='el-icon el-icon-circle-add' style='font-size:20px;'></span>
				</div>
				<div style="height:300px;width:95%;border:1px solid #F3F3F3;">
					<el-ctable ref="ctableFreq" :data="lteForm.NeighFreqList" :pagination="false">
						<el-table-column label="Operations" width="100" prop="" class-name="no-text-tips">
							<template slot-scope="scope">
								<span class="el-icon el-icon-operation-edit" @click="editFreq(scope.row)" style="cursor: pointer;"></span>
								<span class="el-icon el-icon-operation-delete" @click="delFreq(scope.row)" style="margin-left:10px;"></span>
							</template>
						</el-table-column>
						<el-table-column label="Index" prop="Index"></el-table-column>
						<el-table-column label="EARFCN" prop="EARFCN"></el-table-column>
						<el-table-column label="Q-OffsetRange" prop="QOffsetRange"></el-table-column>
						<el-table-column label="qRxLevMinSib5" prop="qRxLevMinSib5"></el-table-column>
						<el-table-column label="PMax" prop="PMax"></el-table-column>
						<el-table-column label="tReselectionEutra" prop="tReselectionEutra"></el-table-column>
						<el-table-column label="ReselThreshHigh" prop="ReselThreshHigh"></el-table-column>
						<el-table-column label="ReselThreshLow" prop="ReselThreshLow"></el-table-column>
						<el-table-column label="ReselectionPriority" prop="ReselectionPriorityFreq"></el-table-column>
						<el-table-column label="Enable" prop="freqEnable" :formatter="enableFmt"></el-table-column>
					</el-ctable>
				</div>
				<el-form-item v-show="false" prop="NeighFreqList" class="validate-item">
					<el-input v-model="lteForm.NeighFreqList"></el-input>
				</el-form-item>

				<div style='width:95%;height:46px;display:flex;margin-top:20px;position: relative;'>
					<span style='font-size:12px;font-weight:bold;flex:1'>
						Neigh Cell List 
						<span v-if="isBLQAndBLN" style="padding-left: 5px;font-weight: normal;color: #999;">LTE neighbor cell can configure up to 160</span>
						<span v-if="!isBLQAndBLN" style="padding-left: 5px;font-weight: normal;color: #999;">LTE neighbor cell can configure up to 128</span> 

						<div v-if="isBaiblq && !isMLQ" class="selectBlukBoxCls" style="display: flex;position: absolute; bottom: 2px;font-weight: normal;">
							<div class="bulkSelectBtnBoxCls" style="display: flex;align-items: center;">
	                            <span class="el-icon-operation-defaultBeta el-icon"></span>
	                            <span class="bulkSelectNumBoxCls">( {{cellSeletions.length}} )</span>
	                        </div>
							<span style="display: inline-block;padding: 0 10px;cursor: pointer;border-left: 1px solid #DFE2EE;margin-left: 5px;" 
								:class="{'disabled': cellSeletions.length==0}"
								@click="batchSetPermanent"> 
								<i class="el-icon el-icon-permanent" style="font-size: 14px;"></i> Permanent
							</span>
							<span style="display: inline-block;padding: 0 10px;cursor: pointer;border-left: 1px solid #DFE2EE;" 
								:class="{'disabled': cellSeletions.length==0}"
								@click="batchSetBlockList"> 
								<i class="el-icon el-icon-operation-blacklist" style="font-size: 14px;"></i> BlockList
							</span>
						</div>
					</span>
					<span v-if="isBLQAndBLN && lteForm.NeighCellList.length < 160" @click='addCell' class='el-icon el-icon-circle-add' style='font-size:20px;line-height:46px;'></span>
					<span v-if="!isBLQAndBLN && lteForm.NeighCellList.length < 128" @click='addCell' class='el-icon el-icon-circle-add' style='font-size:20px;line-height:46px;'></span>
				</div>
				<div style="height:300px;width:95%;border:1px solid #F3F3F3;">
					<el-ctable ref="ctableCell" :data="lteForm.NeighCellList" :pagination="false" row-key="CellIndex" @selection-change="cellSelectionChange">
						<el-table-column type="selection" :selectable="selectable" :reserve-selection="true"></el-table-column>

						<el-table-column label="Operations" width="100" prop="" class-name="no-text-tips">
							<template slot-scope="scope">
								<!-- 黑名单相关 -->
								<span v-if="isBaiblq && [1,'1',3,'3'].includes(scope.row.neighborType)" @click="setPermanent(scope.row)" class="el-icon el-icon-permanent" style="margin-left:10px;"></span>
								<span v-if="isBaiblq && scope.row.neighborType == '1'" @click="setBlockList(scope.row)" class="el-icon el-icon-operation-blacklist" style="margin-left:10px;"></span>
								<!-- 常规操作 -->
								<span v-if="![1,'1',3,'3'].includes(scope.row.neighborType)" class="el-icon el-icon-operation-edit" @click="editCell(scope.row)"  style="cursor: pointer;"></span>
								<span v-if="scope.row.neighborType != '1'" class="el-icon el-icon-operation-delete" @click="delCell(scope.row)" style="margin-left:10px;"></span>
							</template>
						</el-table-column>
						<el-table-column label="Index" prop="CellIndex"></el-table-column>
						<el-table-column label="PLMN" prop="PLMN"></el-table-column>
						<el-table-column v-if="!isBaiblq" label="Cell ID" prop="CellID"></el-table-column>
						<el-table-column label="EARFCN" prop="CellEARFCN"></el-table-column>
						<el-table-column label="PCI" prop="PCI"></el-table-column>
						<el-table-column label="QOffset" prop="QOffset"></el-table-column>
						<el-table-column label="CIO" prop="CIO"></el-table-column>
						<el-table-column label="TAC" prop="TAC"></el-table-column>

						<!-- 黑名单相关字段 -->
						<el-table-column v-if="isBaiblq && !isMLQ" label="ECI" prop="CellID"></el-table-column>
						<el-table-column v-if="isBaiblq && !isMLQ" label="Status" prop="cellEnable">
							<template slot-scope="scope">
								<span v-if="scope.row.cellEnable == 'true'">Active</span>
								<span v-if="scope.row.cellEnable == 'false'">Inactive</span>
							</template>
						</el-table-column>
						<el-table-column v-if="isBaiblq && !isMLQ" label="eNodeB Type" prop="enbType" width="110">
							<template slot-scope="scope">
								<span v-if="['0',0].includes(scope.row.enbType)">Macro</span>
								<span v-if="['1',1].includes(scope.row.enbType)">Home</span>
							</template>
						</el-table-column>
						<el-table-column v-if="isBaiblq && !isMLQ" label="X2 Status" prop="x2Status">
							<template slot-scope="scope">
								<span v-if="scope.row.x2Status == '0'">Not Connection</span>
								<span v-if="scope.row.x2Status == '1'">Connection</span>
							</template>
						</el-table-column>
						<el-table-column v-if="isBaiblq && !isMLQ" label="X2 Flag" prop="x2Flag">
							<template slot-scope="scope">
								<span v-if="['0',0].includes(scope.row.x2Flag)">SON</span>
								<span v-if="['1',1].includes(scope.row.x2Flag)">Manual</span>
							</template>
						</el-table-column>
						<el-table-column v-if="isBaiblq && !isMLQ" label="X2 IP" prop="x2IP"></el-table-column>
						<el-table-column v-if="isBaiblq && !isMLQ" label="Neighbor Type" prop="neighborType" width="110">
							<template slot-scope="scope">
								<span v-if="scope.row.neighborType == '2'">Permanent</span>
								<span v-if="scope.row.neighborType == '1'">ANR</span>
								<span v-if="scope.row.neighborType == '3'">BlockList</span>
							</template>
						</el-table-column>
					</el-ctable>
				</div>
				<el-form-item v-show="false" prop="NeighCellList" class="validate-item">
					<el-input v-model="lteForm.NeighCellList"></el-input>
				</el-form-item>
			</el-collapse-item>
			<el-collapse-item v-show="hasKey('A1ThresholdRSRP')" name='mobility'>
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">Mobility Parameter</span>
					</p>
				</template>
				<p class='item-title-cls'>A1 Event Threshold</p>
				<el-form-item label='A1 Threshold-RSRP' prop="A1ThresholdRSRP" class="validate-item">
					<el-input v-model="lteForm.A1ThresholdRSRP">
						<template slot="append">Range:0~97</template>
					</el-input>
				</el-form-item>
				<p class='item-title-cls'>A2 Event Threshold</p>
				<el-form-item label='A2 Threshold-RSRP' prop="A2ThresholdRSRP" class="validate-item">
					<el-input v-model="lteForm.A2ThresholdRSRP">
						<template slot="append">Range:0~97</template>
					</el-input>
				</el-form-item>
				<p class='item-title-cls'>A3 Event Threshold</p>
				<el-form-item label='A3 Offset' prop="A3Offset" class="validate-item">
					<el-input v-model="lteForm.A3Offset">
						<template slot="append">Range:-30 ~ 30</template>
					</el-input>
				</el-form-item>
				<!--<el-form-item label='A3 Offset ANR' prop="A3OffsetANR" class="validate-item">
					<el-input v-model="lteForm.A3OffsetANR">
						<template slot="append">Range:-30~30</template>
					</el-input>
				</el-form-item>
				<el-form-item label='A3 TimeToTrigger' prop="A3TimeToTrigger" class="validate-item">
					<el-select v-model="lteForm.A3TimeToTrigger">
						<el-option label="ms5120"></el-option>
					</el-select>
				</el-form-item>-->
				
				<!--<p class='item-title-cls'>A4 Event Threshold</p>
				<el-form-item label='A4 Threshold-RSRP' prop="A4ThresholdRSRP" class="validate-item">
					<el-input v-model="lteForm.A4ThresholdRSRP">
						<template slot="append">Range:0~97</template>
					</el-input>
				</el-form-item>-->
				<p class='item-title-cls'>A5 Event Threshold</p>
				<el-form-item label='A5 Threshold1-RSRP' prop="A5Threshold1RSRP" class="validate-item">
					<el-input v-model="lteForm.A5Threshold1RSRP">
						<template slot="append">Range:0~97</template>
					</el-input>
				</el-form-item>
				<el-form-item label='A5 Threshold2-RSRP' prop="A5Threshold2RSRP" class="validate-item error-tips-show">
					<el-input v-model="lteForm.A5Threshold2RSRP">
						<template slot="append">Range:0~97</template>
					</el-input>
				</el-form-item>
				<p class='item-title-cls'>B2 Event Threshold</p>
				<el-form-item label='B2 Threshold1-RSRP' prop="B2RSRPThreshold1" class="validate-item">
					<el-input v-model="lteForm.B2RSRPThreshold1">
						<template slot="append">Range:0~97</template>
					</el-input>
				</el-form-item>
				<el-form-item label='B2 Threshold2-RSRP' prop="B2RSRPThreshold2" class="validate-item">
					<el-input v-model="lteForm.B2RSRPThreshold2">
						<template slot="append">Range:0~97</template>
					</el-input>
				</el-form-item>
				<el-form-item label='GERAN B2 IRAT Threshold' prop="B2IRATThreshold" class="validate-item">
					<el-input v-model="lteForm.B2IRATThreshold">
						<template slot="append">Range:0~63</template>
					</el-input>
				</el-form-item>
				<p class='item-title-cls'>Cell Selection Parameter</p>
				<el-form-item label='Qrxlevmin' class='validate-item unit-item' prop="Qrxlevmin">
					<el-input v-model="lteForm.Qrxlevmin">
						<template slot="append">
							<span class='unit-cls'>dBm</span>
							<span>Range:-70 ~ -22</span>
						</template>
					</el-input>
				</el-form-item>
				<el-form-item label='Qrxlevminoffset' prop="Qrxlevminoffset" class='validate-item'>
					<el-input v-model="lteForm.Qrxlevminoffset">
						<template slot="append">Range:1~8</template>
					</el-input>
				</el-form-item>
				<p class='item-title-cls'>Cell ReSelection Parameter</p>
				<el-form-item label='S-IntraSearch' class='validate-item unit-item' prop="SIntraSearch">
					<el-input v-model="lteForm.SIntraSearch">
						<template slot="append">
							<span class='unit-cls'>dBm</span>
							<span>Range:0~31</span>
						</template>
					</el-input>
				</el-form-item>
				<el-form-item label='QrxlevminSib3' class='validate-item unit-item' prop="QrxlevminSib">
					<el-input v-model="lteForm.QrxlevminSib">
						<template slot="append">
							<span class='unit-cls'>dBm</span>
							<span>Range:-70 ~ -22</span>
						</template>
					</el-input>
				</el-form-item>
				<el-form-item label='ThreshServingLow' prop="ThreshServingLow" class='validate-item'>
					<el-input v-model="lteForm.ThreshServingLow">
						<template slot="append">Range:0~31</template>
					</el-input>
				</el-form-item>
				<el-form-item label='Qhyst' prop="Qhyst">
					<el-select v-model="lteForm.Qhyst">
						<el-option label="dB0" value="0"></el-option>
						<el-option label="dB1" value="1"></el-option>
						<el-option label="dB2" value="2"></el-option>
						<el-option label="dB3" value="3"></el-option>
						<el-option label="dB4" value="4"></el-option>
						<el-option label="dB5" value="5"></el-option>
						<el-option label="dB6" value="6"></el-option>
						<el-option label="dB8" value="8"></el-option>
						<el-option label="dB10" value="10"></el-option>
						<el-option label="dB12" value="12"></el-option>
						<el-option label="dB14" value="14"></el-option>
						<el-option label="dB16" value="16"></el-option>
						<el-option label="dB18" value="18"></el-option>
						<el-option label="dB20" value="20"></el-option>
						<el-option label="dB22" value="22"></el-option>
						<el-option label="dB24" value="24"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label='S-NonIntraSearch' prop="SNonIntraSearch" class='validate-item'>
					<el-input v-model="lteForm.SNonIntraSearch">
						<template slot="append">Range:0~31</template>
					</el-input>
				</el-form-item>
				<el-form-item label='Reselection Priority' prop="ReselectionPriority" class='validate-item'>
					<el-input v-model="lteForm.ReselectionPriority">
						<template slot="append">Range:0~7</template>
					</el-input>
				</el-form-item>
				<el-form-item label='Allowed Meas BW' prop="AllowedMeasBandwidth">
					<el-select v-model="lteForm.AllowedMeasBandwidth">
						<el-option label="CELL_BW_N6(1.4M)" value="n6"></el-option>
						<el-option label="CELL_BW_N15(3M)" value="n15"></el-option>
						<el-option label="CELL_BW_N25(5M)" value="n25"></el-option>
						<el-option label="CELL_BW_N50(10M)" value="n50"></el-option>
						<el-option label="CELL_BW_N75(15M)" value="n75"></el-option>
						<el-option label="CELL_BW_N100(20M)" value="n100"></el-option>
					</el-select>
				</el-form-item>
				<p class='item-title-cls'>X2</p>
				<el-form-item label='X2 Enable' prop="X2Enable">
					<el-switch v-model="lteForm.X2Enable" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
				<p class='item-title-cls'>ANR Parameters</p>
				<el-form-item label='Measurement Configuration' prop="MeasurementCongiguration">
					<el-select v-model="lteForm.MeasurementCongiguration">
						<el-option label="Measurement Disable" value="0"></el-option>
						<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="Intera A3 Event" value="1"></el-option>
						<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="Periodic" value="3"></el-option>
						<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="Intera A5 Event" value="4"></el-option>
						<el-option v-if="isBaiblx_QRTB || isBaiblx_BLQ" label="Event" value="4"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item v-show="lteForm.MeasurementCongiguration == '1' || (lteForm.MeasurementCongiguration == '4' && (isBaiblx_QRTB || isBaiblx_BLQ))" label='Inter-Freq ANR A3 RSRP Threshold' prop="ANRA3RSRPThreshold" class='validate-item'>
					<el-input v-model="lteForm.ANRA3RSRPThreshold">
						<template slot="append">Range:-30~30</template>
					</el-input>
				</el-form-item>
				<el-form-item v-show="lteForm.MeasurementCongiguration == '4'" label='Inter-Freq ANR A5 RSRP Threshold1' prop="ANRA5RSRPThreshold1" class='validate-item'>
					<el-input v-model="lteForm.ANRA5RSRPThreshold1">
						<template slot="append">Range:0~97</template>
					</el-input>
				</el-form-item>
				<el-form-item v-show="lteForm.MeasurementCongiguration == '4'" label='Inter-Freq ANR A5 RSRP Threshold2' prop="ANRA5RSRPThreshold2" class='validate-item'>
					<el-input v-model="lteForm.ANRA5RSRPThreshold2">
						<template slot="append">Range:0~97</template>
					</el-input>
				</el-form-item>
			</el-collapse-item>
			<el-collapse-item v-show="hasKey('TotalTxPower')" name="power">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">Power Control</span>
					</p>
				</template>
				<el-form-item label='Total Tx Power' prop="TotalTxPower" class='validate-item'>
					<el-input v-model="lteForm.TotalTxPower">
						<template slot="append">Range:-30~33</template>
					</el-input>
				</el-form-item>
				<el-form-item label='Po_nominal_pusch' prop="Po_nominal_pusch" class='validate-item'>
					<el-input v-model="lteForm.Po_nominal_pusch">
						<template slot="append">Range:-126~24</template>
					</el-input>
				</el-form-item>
				<el-form-item label='Preamble Init Target Power' prop="PreambleInitTargetPower">
					<el-select v-model="lteForm.PreambleInitTargetPower">
						<el-option label="dBm-120" value="-120"></el-option>
						<el-option label="dBm-118" value="-118"></el-option>
						<el-option label="dBm-116" value="-116"></el-option>
						<el-option label="dBm-114" value="-114"></el-option>
						<el-option label="dBm-112" value="-112"></el-option>
						<el-option label="dBm-110" value="-110"></el-option>
						<el-option label="dBm-108" value="-108"></el-option>
						<el-option label="dBm-106" value="-106"></el-option>
						<el-option label="dBm-104" value="-104"></el-option>
						<el-option label="dBm-102" value="-102"></el-option>
						<el-option label="dBm-100" value="-100"></el-option>
						<el-option label="dBm-98" value="-98"></el-option>
						<el-option label="dBm-96" value="-96"></el-option>
						<el-option label="dBm-94" value="-94"></el-option>
						<el-option label="dBm-92" value="-92"></el-option>
						<el-option label="dBm-90" value="-90"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label='Target ul sinr' prop="Targetulsinr" class='validate-item'>
					<el-input v-model="lteForm.Targetulsinr">
						<template slot="append">Range:-6~10</template>
					</el-input>
				</el-form-item>
				<el-form-item label='PB' prop="PB" class='validate-item'>
					<el-input v-model="lteForm.PB">
						<template slot="append">Range:0~3</template>
					</el-input>
				</el-form-item>
				<el-form-item label='Power Ramping' prop="PowerRamping">
					<el-select v-model="lteForm.PowerRamping">
						<el-option label="0" value="0"></el-option>
						<el-option label="2" value="2"></el-option>
						<el-option label="4" value="4"></el-option>
						<el-option label="6" value="6"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label='Po_nominal_pucch' prop="Po_nominal_pucch" class='validate-item'>
					<el-input v-model="lteForm.Po_nominal_pucch">
						<template slot="append">Range:-127~-96</template>
					</el-input>
				</el-form-item>
				<el-form-item label='Alpha' prop="alpha">
					<el-select v-model="lteForm.alpha">
						<el-option label="0" value="0"></el-option>
						<el-option label="40" value="40"></el-option>
						<el-option label="50" value="50"></el-option>
						<el-option label="60" value="60"></el-option>
						<el-option label="70" value="70"></el-option>
						<el-option label="80" value="80"></el-option>
						<el-option label="90" value="90"></el-option>
						<el-option label="100" value="100"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label='PA' prop="PA">
					<el-select v-model="lteForm.PA">
						<el-option label="-6dB" value="-600"></el-option>
						<el-option label="-4.77dB" value="-477"></el-option>
						<el-option label="-3dB" value="-300"></el-option>
						<el-option label="-1.77dB" value="-177"></el-option>
						<el-option label="0dB" value="0"></el-option>
						<el-option label="1dB" value="100"></el-option>
						<el-option label="2dB" value="200"></el-option>
						<el-option label="3dB" value="300"></el-option>
					</el-select>
				</el-form-item>
			</el-collapse-item>
			<el-collapse-item v-show="hasKey('CipheringAlgorithm')" name="security">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">Security Setting</span>
					</p>
				</template>
				<el-form-item label='Ciphering Algorithm' prop="CipheringAlgorithm">
					<el-select v-model="lteForm.CipheringAlgorithm">
						<el-option label="128-EEA1" value="128-EEA1"></el-option>
						<el-option label="128-EEA2" value="128-EEA2"></el-option>
						<!--<el-option label="128-EEA3" value="128-EEA3"></el-option>-->
						<el-option label="EEA0" value="EEA0"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label='Integrity Algorithm' prop="IntegrityAlgorithm">
					<el-select v-model="lteForm.IntegrityAlgorithm">
						<el-option label="128-EIA1" value="128-EIA1"></el-option>
						<el-option label="128-EIA2" value="128-EIA2"></el-option>
						<!--<el-option label="128-EIA3" value="128-EIA3"></el-option>
						<el-option label="EIA0" value="EIA0"></el-option>-->
					</el-select>
				</el-form-item>
			</el-collapse-item>
			<el-collapse-item name="advance">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">Advance</span>
					</p>
				</template>
				<p v-show="hasKey('ULSchdAlgorithm')" class='item-title-cls'>Scheduling Algorithm</p>
				<el-form-item v-show="hasKey('ULSchdAlgorithm')" label='UL Schd Algorithm' prop="ULSchdAlgorithm" class='validate-item'>
					<el-select v-model="lteForm.ULSchdAlgorithm">
						<el-option label="QoS" value="0"></el-option>
						<el-option label="PFS" value="1"></el-option>
						<el-option label="RR" value="2"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item v-show="hasKey('DLSchdAlgorithm')" label='DL Schd Algorithm' prop="DLSchdAlgorithm" class='validate-item'>
					<el-select v-model="lteForm.DLSchdAlgorithm">
						<el-option label="QoS" value="0"></el-option>
						<el-option label="PFS" value="1"></el-option>
						<el-option label="RR" value="2"></el-option>
					</el-select>
				</el-form-item>
				<p v-show="hasKey('GPSSyncAdjustValue')" class='item-title-cls'>Sync Adjust Parameters</p>
				<el-form-item v-show="hasKey('GPSSyncAdjustValue')" label='GPS Sync Adjust Value' prop="GPSSyncAdjustValue" class='validate-item'>
					<el-input v-model="lteForm.GPSSyncAdjustValue">
						<template slot="append">Range:-65535~65535</template>
					</el-input>
				</el-form-item>
				<el-form-item v-show="hasKey('ICTAAdjustValue')" label='ICTA Adjust Value' prop="ICTAAdjustValue" class='validate-item'>
					<el-input v-model="lteForm.ICTAAdjustValue">
						<template slot="append">Range:-65535~65535</template>
					</el-input>
				</el-form-item>
				<p v-show="hasKey('LinkKeepAlive')" class='item-title-cls'>Link Activation State Detector</p>
				<el-form-item v-show="hasKey('LinkKeepAlive')" label='Link Keep Alive' prop="LinkKeepAlive">
					<el-switch v-model="lteForm.LinkKeepAlive" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
				<el-form-item v-show="hasKey('LinkKeepAliveTimer')" label='Link Keep Alive Timer' prop="LinkKeepAliveTimer">
					<el-select v-model="lteForm.LinkKeepAliveTimer">
						<el-option label="5 Minutes" value="1"></el-option>
						<el-option label="10 Minutes" value="2"></el-option>
						<el-option label="15 Minutes" value="3"></el-option>
					</el-select>
				</el-form-item>
				<p v-show="hasKey('WorkingMode')" class='item-title-cls'>Working Mode</p>
				<el-form-item v-show="hasKey('WorkingMode')" label='Working Mode' prop="WorkingMode" class='validate-item'>
					<el-select v-model="lteForm.WorkingMode">
						<el-option label="32UE" value="32"></el-option>
						<el-option label="64UE" value="64"></el-option>
						<el-option label="96UE" value="96"></el-option>
						<el-option v-if="isBaiblq || isBaiblx_QRTB || isBaiblx_BLQ" label="128UE" value="128"></el-option> <!-- #72093 -->
						<el-option v-if="isBaiblq || isBaiblx_BLQ" label="256UE" value="256"></el-option>
					</el-select>
				</el-form-item>
				
				<p v-show="hasKey('ZeroConfig')" class='item-title-cls'>Random Access Parameters</p>
				<el-form-item v-show="hasKey('ZeroConfig')" label='Zero Correlation Zone Config' prop="ZeroConfig" class='validate-item'>
					<el-input v-model="lteForm.ZeroConfig">
						<template v-if="!isBaiblq" slot="append">Range:0~94</template>
						<template v-if="isBaiblq" slot="append">Range:0~13</template>
					</el-input>
				</el-form-item>
				<el-form-item v-show="hasKey('rootSequenceIndex1')" :label="(isBaiblq || isBaiblx_BLQ ||isBaiblx_QRTB)?'rootSequenceIndex':'rootSequenceIndex1'" prop="rootSequenceIndex1" class='validate-item'>
					<el-input v-model="lteForm.rootSequenceIndex1">
						<template slot="append">Range:0~837</template>
					</el-input>
				</el-form-item>
				<el-form-item v-if="!(isBaiblq || isBaiblx_BLQ ||isBaiblx_QRTB) && hasKey('rootSequenceIndex2')" label='rootSequenceIndex2' prop="rootSequenceIndex2" class='validate-item'>
					<el-input v-model="lteForm.rootSequenceIndex2">
						<template slot="append">Range:0~837</template>
					</el-input>
				</el-form-item>
				<el-form-item v-show="hasKey('PRACHFreqOffset')" label='PRACH Freq Offset' prop="PRACHFreqOffset" class='validate-item'>
					<el-input v-model="lteForm.PRACHFreqOffset">
						<template v-if="!isBaiblq" slot="append">Range:0~94</template>
						<template v-if="isBaiblq" slot="append">Range:3~16</template>
					</el-input>
				</el-form-item>
				<el-form-item v-show="hasKey('configurationIndex')" label='configurationIndex' prop="configurationIndex" class='validate-item'>
					<el-input v-model="lteForm.configurationIndex">
						<template v-if="!isBaiblq" slot="append">Range:0~837</template>
						<template v-if="isBaiblq" slot="append">Range:0~15</template>
					</el-input>
				</el-form-item>
			</el-collapse-item>
		</el-collapse>
	</el-form>
	<div class='addSlide' id='neighPanel' v-show="showNeigh">
		
	</div>
</div>
<script>
	var lteVm = new Vue({
		el:'#ltePanel',
		data(){
			var validateRange = (rule,value,callback)=>{
					var min = rule.min;
					var max = rule.max;
					var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
					if(value == '' || (reg.test(value) && value >= min && value <= max)){
						callback();
					}else{
						callback(new Error('format error'))
					}
				},
				validateBLXA5 = (rule,value,callback) => {
					var anrA5 = this.lteForm.ANRA5RSRPThreshold2,
						isEventType = this.lteForm.MeasurementCongiguration == '4';

					if(isEventType && anrA5 && (value - anrA5 < 0)) {
						callback('Should be greater than or equal to "Inter ANR A5 RSRP THRESHOLD2"('+ anrA5 +')')
					}else {
						callback()
					}
				};
			var validatePlmn = (rule,value,callback) => {
				var minVal = rule.min;
				var maxVal = rule.max;
				var reg = /^-?\d+$/;
				if(value == ""){
					callback();
				}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal)) && value.length<=6 && value.length >=5){
					callback();
				}else{
					callback(new Error('format error'));
				}
			}
			var isBaiBLQ = settingVue.selectedRow.product == 'BAIBLQ' || settingVue.selectedRow.product == 'MLQ';
			var isMLQ = settingVue.selectedRow.product == 'MLQ';
			var isBLQAndBLN = ['MLN-CA','MLN-DC','MLN-SC','BAIBLQ'].includes(settingVue.selectedRow.product);
			
			return {
				isBaiblx_QRTB: settingVue.selectedRow.platformType == 'BLX' && settingVue.selectedRow.network_model == 'TDDMode',
				isBaiblx_BLQ: settingVue.selectedRow.platformType == 'BLX' && settingVue.selectedRow.network_model == 'FDDMode',

				isBLQAndBLN: isBLQAndBLN || isBaiBLQ,

				isBaiblq: isBaiBLQ,
				isMLQ: isMLQ,
				codeList: [],

				rebootMap: {},

				originNeigh: {
					cell: [],
					freq: []
				},
				delRecord: {
					NeighFreqList: [],
					NeighCellList: []
				},
				codeTableList: [],
				lteForm:{
					NeighFreqList:[],
					NeighCellList:[],
					A1ThresholdRSRP:'',
					A2ThresholdRSRP:'',
					A3Offset:'',
					//A3OffsetANR:'',
					//A4ThresholdRSRP:'',
					A5Threshold1RSRP:'',
					A5Threshold2RSRP:'',
					Qrxlevmin:'',
					Qrxlevminoffset:'',
					B2RSRPThreshold1:'',
					B2RSRPThreshold2:'',
					B2IRATThreshold:'',
					SIntraSearch:'',
					ThreshServingLow:'',
					QrxlevminSib:'',
					Qhyst:'',
					SNonIntraSearch:'',
					ReselectionPriority:'',
					AllowedMeasBandwidth:'',
					X2Enable:'true',
					MeasurementCongiguration:'',
					ANRA3RSRPThreshold: '',
					ANRA5RSRPThreshold1:'',
					ANRA5RSRPThreshold2:'',
					TotalTxPower:'',
					Po_nominal_pusch:'',
					Po_nominal_pucch:'',
					PreambleInitTargetPower:'',
					Targetulsinr:'',
					PB:'',
					PowerRamping:'',
					alpha:'',
					PA:'',
					CipheringAlgorithm:'',
					IntegrityAlgorithm:'',
					ULSchdAlgorithm:'',
					DLSchdAlgorithm:'',
					GPSSyncAdjustValue:'',
					ICTAAdjustValue:'',
					LinkKeepAlive:'',
					LinkKeepAliveTimer:'',
					WorkingMode:'',
					// MOCNEnable:'',
					// isPrimary:'',
					// PLMN:'',
					ZeroConfig:'',
					rootSequenceIndex1:'',
					rootSequenceIndex2:'',
					PRACHFreqOffset:'',
					configurationIndex:''
				},
				lteRules:{
					A1ThresholdRSRP:[{validator:validateRange,min:0,max:97}],
					A2ThresholdRSRP:[{validator:validateRange,min:0,max:97}],
					A3Offset:[{validator:validateRange,min:-30,max:30}],
					//A3OffsetANR:[{validator:validateRange,min:-30,max:30}],
					//A4ThresholdRSRP:[{validator:validateRange,min:0,max:97}],
					A5Threshold1RSRP:[{validator:validateRange,min:0,max:97}],
					A5Threshold2RSRP:[
						{validator: validateRange,min:0,max:97},
						{validator: validateBLXA5},
					],
					Qrxlevmin:[{validator:validateRange,min:-70,max:-22}],
					Qrxlevminoffset:[{validator:validateRange,min:1,max:8}],
					B2RSRPThreshold1:[{validator:validateRange,min:0,max:97}],
					B2RSRPThreshold2:[{validator:validateRange,min:0,max:97}],
					B2IRATThreshold:[{validator:validateRange,min:0,max:63}],
					SIntraSearch:[{validator:validateRange,min:0,max:31}],
					ThreshServingLow:[{validator:validateRange,min:0,max:31}],
					QrxlevminSib:[{validator:validateRange,min:-70,max:-22}],
					SNonIntraSearch:[{validator:validateRange,min:0,max:31}],
					ReselectionPriority:[{validator:validateRange,min:0,max:7}],
					ANRA3RSRPThreshold:[{validator:validateRange,min:-30,max:30}],
					ANRA5RSRPThreshold1:[{validator:validateRange,min:0,max:97}],
					ANRA5RSRPThreshold2:[{validator:validateRange,min:0,max:97}],
					TotalTxPower:[{validator:validateRange,min:-30,max:33}],
					Po_nominal_pusch:[{validator:validateRange,min:-126,max:24}],
					Targetulsinr:[{validator:validateRange,min:-6,max:10}],
					PB:[{validator:validateRange,min:0,max:3}],
					Po_nominal_pucch:[{validator:validateRange,min:-127,max:-96}],
					ULSchdAlgorithm:[{validator:validateRange,min:0,max:2}],
					DLSchdAlgorithm:[{validator:validateRange,min:0,max:2}],
					GPSSyncAdjustValue:[{validator:validateRange,min:-65535,max:65535}],
					ICTAAdjustValue:[{validator:validateRange,min:-65535,max:65535}],
					WorkingMode:[{validator:validateRange,min:-1,max:256}],
					//PLMN:[{validator:validatePlmn,min:00000,max:999999}],
					ZeroConfig:[{validator:validateRange,min:0,max: isBaiBLQ?13:94}],
					rootSequenceIndex1:[{validator:validateRange,min:0,max:837}],
					rootSequenceIndex2:[{validator:validateRange,min:-1,max:837}],
					PRACHFreqOffset:[{validator:validateRange,min:isBaiBLQ?3:0,max:isBaiBLQ?16:94}],
					configurationIndex:[{validator:validateRange,min:0,max:isBaiBLQ?15:837}],
				},
				cellData:[],
				showNeigh:false,
				activeNames:['neigh','mobility','power','security','advance'],
				casts:{
					'EE65D67A83D4DC154D11F65E8820G853':'NeighFreqList',
					'17E3837E1889C2BBB19926BAEA4E8DC9':'freqEnable',
					'0EE9F73309A62895413D4A6AA017E748':'Index',
					'FE483865456C8B7A7FEB5F04B6D67812':'EARFCN',
					'034A3AB30A3CEB51152B7CF644F824DA':'QOffsetRange',
					'C204695B05E8A0F2D475EF341EA4E17C':'qRxLevMinSib5',
					'149B0706B83E974841EDD83A5DD89028':'PMax',
					'60582B76633ED5CB6004BC99FECEC939':'tReselectionEutra',
					'820A7ABCEE60F42885AAD9D13F05025D':'ReselThreshHigh',
					'8DB412E0236AA1342B70D239AF35C344':'ReselThreshLow',
					'1128338A3D336947EBEF916C14A29BE8':'ReselectionPriorityFreq',
					/* cell */
					'787DFEF7B9383D1BB3E8DFC50311691G':'cellEnable',
					'CF50CE9840B56513B9CF7CC3E4FC7A66':'NeighCellList',
					'BBA865D7D0F1D9400D87435895A5C54C':'CellIndex',
					'932D9B4C821B99EE924CABD169E1G7B1':'CellID',
					'94A9445BDDF78ACB007379431AD1F6C9':'PLMN',
					'0FD954E8679A6E650D7F55454E8638F1':'CellEARFCN',
					'94118211DBF350DBB2E8143ABA5BCACC':'PCI',
					'C852FCB7C355F2D0674442737FA0117E':'QOffset',
					'EE146A1DA46630B9064068ECE7487FCD':'CIO',
					'E375EFB542BEC7F8926164F9CAA3BC94':'TAC',
					/* Mobility Parameter */
					'9A0BE6050E9048FCD89C22FB70C5CE30':'A1ThresholdRSRP',
					'F48DC2126118A909CD711A6A784092BE':'A2ThresholdRSRP',
					'62AA5B190BA85AEB6ABF7B1CB5F6C4BB':'A3Offset',
					//'09725516F94CEB39E57186ECD4DF19F2':'A3OffsetANR',
					//'98A81D6D21933D67FC2A738D71A16D40':'A4ThresholdRSRP',
					'A9342DD4FEA6D05AF6FF904F734C7493':'A5Threshold1RSRP',
					'3864A8CE3CC5121644135950A34F3400':'A5Threshold2RSRP',
					'10F7ED03D0EDCBACD864EAF0DD4F8B84':'B2RSRPThreshold1',
					'8A66DB6D8966417CA5779EA9789EB88C':'B2RSRPThreshold2',
					'58094B9747BC4CE91C02531071796846':'B2IRATThreshold',
					'89233E6077AA3249D22979CB039FCB4E':'Qrxlevmin',
					'DFEB92C95B6228A81E5A6E32257B0B48':'Qrxlevminoffset',
					'409973E97E79BA34B304E9309E2B026D':'SIntraSearch',
					'F9B355DDFB6C23473A76926BD745F93F':'ThreshServingLow',
					'B165AEAB9629022ACAD52CF3BBF48B6F':'QrxlevminSib',
					'1411B82B01FE21BC13332B93689E34E6':'Qhyst',
					'DE5C3EE5ABA1A2B57EEDC5ACE7122748':'SNonIntraSearch',
					'89438F44C791E1C13E1528BA006E9A8A':'ReselectionPriority',
					'D0654D5F79ED8943AAC9A1AB2A18E6F6':'AllowedMeasBandwidth',
					'059876B809EEF57103635E117218E4AE':'X2Enable',
					'4D084C2A2016F36AA03B1600FCAC1E24':'MeasurementCongiguration',
					'D52902A0FCB332BBA002905B36B66069':'ANRA3RSRPThreshold',
					'D9053AAD05AAD6E3E1D9AFB552C281F0':'ANRA5RSRPThreshold1',
					'967DCE641591D79813D31A1EE37F879A':'ANRA5RSRPThreshold2',
					/* power control */
					'0C736C3C0FF84122EC4E99C0F1E51232':'TotalTxPower',
					'468664E1F70EF0FC6F6D8F1A7BB16D01':'Po_nominal_pusch',
					'EEC671896F3B17F7D4AEA5AC6DC0A7A3':'Po_nominal_pucch',
					'AC68D66AD0ACD40C812B29020C507646':'PreambleInitTargetPower',
					'5EA5311DBEFC2F809A9CF6276772732B':'Targetulsinr',
					'32FD6521655C47F9B9EAB5F5D0CB1CA4':'PB',
					'60FE77CCB6A2AA918778AC24CB56C68B':'PowerRamping',
					'7EF76C21D9B1F31D66555BE9F89CA350':'alpha',
					'CF1327F21A63E8380B75280F06EB20E6':'PA',
					'AAD69E21495B86AF22D075CC90E7B4C6':'CipheringAlgorithm',
					'79894C4C5EE336B234F4F219FCDEBB34':'IntegrityAlgorithm',
					/* advance */
					'5E7FE1D5E1F47D65CC60E13E7D68A126':'ULSchdAlgorithm',
					'E7D05F06AC0755DF0F39FA78DE371795':'DLSchdAlgorithm',
					'BDC8BFC0CECCC3C61BCE4CF3FFF2585E':'GPSSyncAdjustValue',
					'629A11D8CF9FB2CBFAE5B91C5A684773':'ICTAAdjustValue',
					'5141DC38CC0C45B44F3429143D7066AD':'LinkKeepAlive',
					'9F44CB818555E7F9A053AD93C12CD556':'LinkKeepAliveTimer',
					'B8CF3AA41ABF532080B3F42F6E0E3915':'WorkingMode',
					'39F5020EF8BC948A49CF3B76479EC1EB':'MOCNEnable',
					'8BBC70AD0906311CDC5F9B2DBCD4FFB7':'isPrimary',
					'B706EAD23D04C0F70B94C0941B8C6064':'PLMN',
					'4ABAA2807189CD041DC029CA10771EE6':'ZeroConfig',
					'AA8C66C54EBD39D00CDD964097FFCF2B':'rootSequenceIndex1',
					'9839B8CBD7E8A4107C2016932B03282F':'rootSequenceIndex2',
					'5EB38110D003D75A9204D84D771AF668':'PRACHFreqOffset',
					'14FD17E682BE51B0571E5C17F8DBBD82':'configurationIndex',
					
					'66E2A9F43B74EE16A989CA9A8B8C4EFD':'NeighFreqList',
					'DFCCB30D658534711BC36C452337E554':'freqEnable',
					'B132B64E1D77E5AE949B39FD626A4091':'Index',
					'720008E45313A6E7244A297842786E35':'EARFCN',
					'00E70C245863DD922EC41B3C0D17AF94':'QOffsetRange',
					'9B89AAC665C447F66440FD0F741DFFCA':'qRxLevMinSib5',
					'0CBE97DB56A5D3368DFAF7AF63B6917F':'PMax',
					'BA3E54860E4D410E54EEA9AC097F6C40':'tReselectionEutra',
					'C8A80889B261E993BE3EF49928B46F60':'ReselThreshHigh',
					'0BAA00713341A5B19B41B8BEA8B31ED3':'ReselThreshLow',
					'818CD84472AF4BDAE3EA437A4770FE18':'ReselectionPriorityFreq',
					/* cell */
					'A3572D7A3D2F41DE2D374A4A3EACDDA0':'cellEnable',
					'11C18253951DE5980D8BC242DA673F73':'NeighCellList',
					'8D3C0713E7D37FD5AA900AB085EC8596':'CellIndex',
					'014D56464FA684BD02695D29D203223F':'CellID',
					'FA6F8A8AE5F5E8F0203637C02B9855B2':'PLMN',
					'5774739D6C8EA80A57BDBC2637B823FB':'CellEARFCN',
					'37A4D64AAAA313B71D6F09C4F3A4C231':'PCI',
					'28EE20508167A460BC374E39743B7A66':'QOffset',
					'EE4E3475C002C29F0AD167C1332B720A':'CIO',
					'F55857CCAF046F771BDDCCB7C21CC033':'TAC',
					/* Mobility Parameter */
					'F2740BB0AEA186D8ECB4835483333D30':'A1ThresholdRSRP',
					'561F31186A54A0759C7D4B0D7C651C19':'A2ThresholdRSRP',
					'40A1E0DBF183F6BDEB453D83F47D7077':'A3Offset',
					//'09725516F94CEB39E57186ECD4DF19F2':'A3OffsetANR',
					//'98A81D6D21933D67FC2A738D71A16D40':'A4ThresholdRSRP',
					'13EC1432636C8AEBE7EBDF9C7AB8F67D':'A5Threshold1RSRP',
					'F7A4F99D29F90F04CF261700249D1E46':'A5Threshold2RSRP',
					'25763AF0F1095531FBE86010E53B9510':'B2RSRPThreshold1',
					'89451059F538A88BD52497A58DC75295':'B2RSRPThreshold2',
					'D39D8783643A4F502D618D7375597161':'B2IRATThreshold',
					'4BB23B3A9D146D23CAE25C6B72C0D168':'Qrxlevmin',
					'E5D2D2565C9430B49333A0F7D0DF6C6F':'Qrxlevminoffset',
					'46169DE14226E7E430F3B64A9EC04E25':'SIntraSearch',
					'30286CDFE2F0B82B3A7AA510BCFF8083':'ThreshServingLow',
					'1B11FB9B0CAB56336D75076810A4BF41':'QrxlevminSib',
					'89C3A347B7ABE53156A864FFE3E81D19':'Qhyst',
					'C1C04DEABB76356DC508C07F658FF082':'SNonIntraSearch',
					'3EB41B7643F688525AED837F11311096':'ReselectionPriority',
					'5A2BB1AAD5350AE4071EDB00C7988532':'AllowedMeasBandwidth',
					'2F3066D3A233772B0030F689CF797BD1':'X2Enable',
					'309D1E3F2669A6964D7F413602A3667E':'MeasurementCongiguration',
					'F09F271A36D70624C86703D80BB9585D':'ANRA3RSRPThreshold',
					'433FCBC2A294D8A01715A6A1904D52CC':'ANRA5RSRPThreshold1',
					'48D56F53084B2BDB52C6CB8C2CB0E7F9':'ANRA5RSRPThreshold2',
					/* power control */
					'3BF08FC2ED106BEB5661CBBC8EE7E495':'TotalTxPower',
					'E0AA2999A4AC5EF2556C4A4B185B3C4B':'Po_nominal_pusch',
					'D9C13801036D947857A3DD8DB55FDAED':'Po_nominal_pucch',
					'EC42C7AE52CB9B764AC468FC232C58CF':'PreambleInitTargetPower',
					'7DD1795797CCCF8BAE78391B8DC29733':'Targetulsinr',
					'F1D1125AB674FA77375B77E8E9206D22':'PB',
					'42BEE6BAEC17F3542BEF17CFDEF7621A':'PowerRamping',
					'6C201ED72E3B39F815F6FE69D2ABF556':'alpha',
					'D31262AD2E1B4DA0BF7D0180E53E5C4E':'PA',
					'230FEC1334C364E1376B0F3EFD59946A':'CipheringAlgorithm',
					'E5F017CBCBD52BEFBA4A33CD9EB3F566':'IntegrityAlgorithm',
					/* advance */
					'8F8AF299E3B4966A76523F22D8A06433':'ULSchdAlgorithm',
					'E5FC3285CDEDDE6DCC30431212AA344E':'DLSchdAlgorithm',
					'2F7E3FFA425F522AD68269F62A5077E6':'GPSSyncAdjustValue',
					'39DA6AE851F2D3890D32FFF7DF9645D1':'ICTAAdjustValue',
					'8C2F6E3EDAED610639BBE419A596A1BE':'LinkKeepAlive',
					'B8321AEE1C06CFE47E34DF608EC76FFC':'LinkKeepAliveTimer',
					'77DB5C59EDD59CD01559CE6C2373DB83':'WorkingMode',
					'39F5020EF8BC948A49CF3B76479EC1EB':'MOCNEnable',
					'8BBC70AD0906311CDC5F9B2DBCD4FFB7':'isPrimary',
					'B706EAD23D04C0F70B94C0941B8C6064':'PLMN',
					'CC6213C4A3959DE20885D95D4151CEE4':'ZeroConfig',
					'79D8D0FEE9C34D6E96BBCB4FA4E2471D':'rootSequenceIndex1',
					'F52ADB689F9D4AC66EAC153BD744D5C3':'rootSequenceIndex2',
					'F6E11A4BFFE97E46ADC44772D40879B9':'PRACHFreqOffset',
					'6D600804C5E60A9AE8E3E3C5262C5FB8':'configurationIndex',
					
					'EE89FEC80064EF4C0C94E1DD8B99ACA2':'NeighFreqList',
					'5A8C57B862E936BD436908EEC35A6500':'freqEnable',
					'9B08A17C6CB303A526D13BD169097635':'Index',
					'88E4375658954BAC8A3F5D4685331029':'EARFCN',
					'5E493A268DB3C28CD47435945D524EE9':'QOffsetRange',
					'E56E52217AC50293656E03B96A292065':'qRxLevMinSib5',
					'F2316E3751F680E2491A04156D8ECF6A':'PMax',
					'4969EC5E74F7CA2687122CC459BCFFC7':'tReselectionEutra',
					'CB874B40648EF1FE2845993FF5925E87':'ReselThreshHigh',
					'AB034E2912AF6ED43E883D50B3D604A9':'ReselThreshLow',
					'29E2074237DADA7E34AA5E646C890DD9':'ReselectionPriorityFreq',
					/* cell */
					'B5780E8559566D9CBCCDAD15D9F0CC32':'cellEnable',
					'C045BBEAC212E762B200276011D41830':'NeighCellList',
					'2875AC0880D0493B6EC79D646670E6D8':'CellIndex',
					'1ECCF2524FB911F152B2441E641269D0':'CellID',
					'B7FE38F6FA5C8DEEF4725646050A7219':'PLMN',
					'64ECB8CD0FED7F0E018797907CFF0C86':'CellEARFCN',
					'7E725C770B1095F3A63B4169307CE266':'PCI',
					'FEE18581564123E47C740F9226981A5E':'QOffset',
					'B98C3380248DF52B27DBD5E3CD4ACBCD':'CIO',
					'2B8CBF9A2C7A59FBDDBA978AE3555774':'TAC',
					/* Mobility Parameter */
					'B583F50D321C8E6E035E0E667B1870B9':'A1ThresholdRSRP',
					'DEADDFAFC16C6D745019B226C0420DB4':'A2ThresholdRSRP',
					'9D840FC6EB73491D29D1D73F363D9551':'A3Offset',
					//'09725516F94CEB39E57186ECD4DF19F2':'A3OffsetANR',
					//'98A81D6D21933D67FC2A738D71A16D40':'A4ThresholdRSRP',
					'5CF11CB192A35D1FB2CF4BC3ACF1F846':'A5Threshold1RSRP',
					'AE95667F3012EFACDBDE076EC73A6B32':'A5Threshold2RSRP',
					'54F09FA1909E5744888653357772BD64':'B2RSRPThreshold1',
					'04BDDF3ADDA043F04D9950F5C89DE91F':'B2RSRPThreshold2',
					'70BE6661742636D51193F420278D2DC8':'B2IRATThreshold',
					'768A076C2EDBEC38F5F6D40B513A1608':'Qrxlevmin',
					'2C7D2599B51D69EBA5233E345127EFE4':'Qrxlevminoffset',
					'16656EEBD474DCF211EE6A84C09A9BD7':'SIntraSearch',
					'FBA35856D59C4E5DD3E1906351D5F33E':'ThreshServingLow',
					'FDE6213653239DE6577FA3551558DF85':'QrxlevminSib',
					'84B6C6521EF54F30BA87B5C0C1D70BD0':'Qhyst',
					'76EFAE311A8158138B558D568FFC3E31':'SNonIntraSearch',
					'D7923D9EEA1B090D7CE9BE0CDB88BEBC':'ReselectionPriority',
					'FF0E285DF38C19893FFB31A26E33F9BB':'AllowedMeasBandwidth',
					'5E65401EEEE5A557C8DAF84BC706633E':'X2Enable',
					'274A2FBB37ADFB79B341F39C4766A379':'MeasurementCongiguration',
					'90B629BA0792D1CFF6A60EFE4B3B6BD2':'ANRA3RSRPThreshold',
					'1568FCB21936B58D2974AF01B080A262':'ANRA5RSRPThreshold1',
					'2D4DA8DC233EE4BDA0FAF4183008F98A':'ANRA5RSRPThreshold2',
					/* power control */
					'E8AA48F25718D9D592AA5B25D8518133':'TotalTxPower',
					'C0D807EEE3905E1BFC8B2C85B277AA76':'Po_nominal_pusch',
					'F7FF634F07F751E5D5777C3D76B3A8B0':'Po_nominal_pucch',
					'75A0B42316CEAC71D374621E6FEF3ABF':'PreambleInitTargetPower',
					'3CCFDFC6BDACCA78299995A0E11919AE':'Targetulsinr',
					'DF79820774E41C09A32E2164B438B41F':'PB',
					'CED5FBC474AC89D868D4301DB6EAC4B2':'PowerRamping',
					'F4686B94525D97597663C453615AA6C8':'alpha',
					'998A5D5F578472F0705A0F922C4FCC1C':'PA',
					'12690FAA8AE168421F087FEEC7DC9095':'CipheringAlgorithm',
					'114BD0AF612427AF966330B43114DA30':'IntegrityAlgorithm',
					/* advance */
					'92D0C2D999A5BB6716ABC9CAB92DE2AF':'ULSchdAlgorithm',
					'FD59390AD5005AC73B3740D35271A25B':'DLSchdAlgorithm',
					'7B1C8E94CA83DD03D637C234C028857D':'GPSSyncAdjustValue',
					'CFE2E75189FFA41066C0700C4E612593':'ICTAAdjustValue',
					'91D84AC264232C31283522A2124A5E44':'LinkKeepAlive',
					'57A7EBF52ED6ADD36C2EC7B618A50A19':'LinkKeepAliveTimer',
					'5201DA64A46482457862715BF482BD11':'WorkingMode',
					'39F5020EF8BC948A49CF3B76479EC1EB':'MOCNEnable',
					'8BBC70AD0906311CDC5F9B2DBCD4FFB7':'isPrimary',
					'B706EAD23D04C0F70B94C0941B8C6064':'PLMN',
					'7ACFA2AD6164AED5338FDA87F362B4CD':'ZeroConfig',
					'0745EEB5020574A7EC8FAA448CF25ABC':'rootSequenceIndex1',
					'A0E501331D2C5C6332EB55A3B73388C6':'rootSequenceIndex2',
					'37C6050912495376DA82348F05D7EF45':'PRACHFreqOffset',
					'BAC0E404188825B867B3F26B5A4BC077':'configurationIndex',

					// BaiBLQ_436Q DC cell2
					'3AD27B21B9FE689216C905A8FE659035':'NeighFreqList',
					'FCD278906DB4BD140178DDCFF27EDD0B':'freqEnable',
					'90390598EC3E0DEA9C8DD3A3147956EF':'Index',
					'2E1D648DA956EB0CABD7B8452704EF5B':'EARFCN',
					'3848C29622611C0A0F9DA82B23ACCCFE':'QOffsetRange',
					'8FAB804CDF7B68B5B147F311B4003CF5':'qRxLevMinSib5',
					'DCF0B7EA009B7BD8113E7F088BDDF23A':'PMax',
					'17F5CC6550927B031682E6B184DDCBE7':'tReselectionEutra',
					'231CB01232F4126584B147602640EED7':'ReselThreshHigh',
					'970DB2FC74D45F062A71279583821E68':'ReselThreshLow',
					'7E3B85C2995076B2CABE32B4974DEEB1':'ReselectionPriorityFreq',
					/* cell */
					'6218BABD8711ADF0F1B0657313FCC22E':'cellEnable',
					'D8C4BE80A9ACD78C5B2BD01135F2DFFF':'NeighCellList',
					'25CC68C80D07F173E588BC391D17E163':'CellIndex',
					'ADC8FD35C162E627A7E6F2DB4DDD8A23':'CellID',
					'1E379F207A1762DB1AC17AD16CB4AE62':'PLMN',
					'279E2AAB17984E48409A8C4D134EB4BA':'CellEARFCN',
					'466E8727AE7F763DB6AABC82A2136DAA':'PCI',
					'3304D30AF4AE41B8D81EC7ED5ECEBB12':'QOffset',
					'1BCE6A76CD97206C65FC36376E5A9B3C':'CIO',
					'1621F0330306C0010651107BE2064FC0':'TAC',
					
					'C02215B52A39EE68452F4ACBF5BA3B83': 'enbType',
					'DCEA103F96B12712DD7DCDB992A1410E': 'x2Flag',
					'5051DDA4DD1BA98D83BDF3BD60859CA5': 'x2IP',
					'5C53DAC7A1EC5B708BB508776805EA97': 'neighborType',
					//'A3572D7A3D2F41DE2D374A4A3EACDDA0': 'status', // cellEnable
					'F03F4A86F2B2FB3699DAF247A075A266': 'x2Status',

					// o6
					'FFB567AC1172CE69D275BA4FB379C995': 'NeighFreqList',
					'6CE2A6F58B2762550791617873A344BA':'freqEnable',
					'6C37504636F41A0AC6AFEC39E8C1F2EA':'Index',
					'BC0CDE1DCC32ECE2D202661CBCEF8ADD':'EARFCN',
					'460103B00B628EE68031C4AD5E7EB78A':'QOffsetRange',
					'AA9E57A627985439B1A8F78CFFF6F5FF':'qRxLevMinSib5',
					'D598D3FBE0893E28829DF05E3527AFC3':'PMax',
					'8FA3E94CC6926BAB6B9EFD2CF6F7D345':'tReselectionEutra',
					'D5397CACA080734D0B86A9DB88230CE5':'ReselThreshHigh',
					'3A71AAE9676AF8B9BF2943CC4BE38FB0':'ReselThreshLow',
					'7AFC02A48107EA30EEA0C90FE3311BA8':'ReselectionPriorityFreq',

					'A299363C05065FFB147A22A2BBCB948E':'cellEnable',
					'279BCAF98B3C72E83DB2E3FA1677EDE0': 'NeighCellList',
					'7F29D2EC7766D09A7E750F650E9F74BF':'CellIndex',
					'BB1DA9B72F03359A46BC4352B8EEC47C':'CellID',
					'F2BD8EFFCFA22056B4E93CC82861944A':'PLMN',
					'94FDEE1EBDA03A28BB32478501692647':'CellEARFCN',
					'BA3FC90A8402420F022191B63631211F':'PCI',
					'634883180C0E1844ED4ABCC3E625872E':'QOffset',
					'2DC8DA15BA79937BDDF9A6C430345EF4':'CIO',
					'33E9FB57754E24A1A35ED260558C63FB':'TAC',

					'04519457D4022E9248C88EC916D7B2EF': 'A1ThresholdRSRP',
					'0DCDA18E1B3A98E697E612FEA321ECE1': 'A2ThresholdRSRP',
					'76BA630308DCFC32EFB8A66E85B8F748': 'A3Offset',
					'A9E5D9A5788019B2C32DE46CDE899D59': 'A5Threshold1RSRP',
					'C2E0F9B711062C788471DC6C34341B11': 'A5Threshold2RSRP',
					'B70BA81B4E09FC77F193AEAE5AB1FFB7': 'ANRA5RSRPThreshold1',
					'F86E8A5C5A284EEFEC0157280ED75E8D': 'ANRA5RSRPThreshold2',
					'69B9508AB11D260C33A77CEBE73C6D37': 'B2RSRPThreshold1',
					'19D5B1E2B9A111427F5A95B9617D0443': 'B2RSRPThreshold2',
					'760B4E5A342278038AA34E93780A8892': 'B2IRATThreshold',
					'F1DEE4FF8A35B9D1701E65C36A0EF0D8': 'Qrxlevmin', 
					'C11BCF432EC12D961465C87D778B9557': 'SIntraSearch',
					'4E9D75EF4C8F64DABFB04AFEC6FB8495': 'QrxlevminSib', 
					'0019A5D7FDEFFBBB53CBC2187882A2A8': 'ReselectionPriority',
					'D4B12DA7552AB680394E97C7DF08F589': 'AllowedMeasBandwidth',
					'B9A7B924B445CB8DEFC00A51E792AF28': 'X2Enable',
					'C840EE1038900FF51122A49CE0D6DE6D': 'MeasurementCongiguration',
					'DE34BD4A339A2DE43AE0110E08E7E92B': 'ANRA5RSRPThreshold1',
					'4A2DA7975066CC1BAFDC85591C7EEB0F': 'ANRA5RSRPThreshold2',
					'2236231B36C8303AF2D5D15098C6806A': 'ANRA3RSRPThreshold',
					'CCDE4E66A57329BB977C7F2B4E7C5CD4': 'TotalTxPower',
					'923DC4A30F3E3173B95BEC5A3133432D': 'PreambleInitTargetPower',
					'DE492913EBE91B0974FBB3DB385366C2': 'Po_nominal_pusch',
					'956443DBE2A4DE461A9DCA02007517D2': 'Po_nominal_pucch',

					'FB6EFC75141499202E4B39B0111D9104': 'A3TimeToTrigger', // 无dom
					'C45198DBFD6F6C5377937AB8DFDA5BF4': 'Qrxlevminoffset',
					'C1A525B009CC8D4F015A8527EF723BA6': 'PowerRamping',
					'A48553D3664D80BD95F828B8810E5BBA': 'ThreshServingLow',
					'8FE6026E8ED3D91FCA76E9C5336AFD64': 'alpha',
					'011E1CD67B38F65D532E172CE0C4FEF6': 'LinkKeepAliveTimer',
					'09C62AC397E16A3967430144A6B7FD9A': 'PB',
					'113B3A7415C77FC59EAB45B716C2E2C7': 'CipheringAlgorithm',
					'1A6BF292F896A597679F134FDF68C4DF': 'rootSequenceIndex2',
					'29C312655B4AFD306EA76FC02C4FEF4C': 'IntegrityAlgorithm',
					'38EFB74C117FFE9388DE599C91DD4655': 'GPSSyncAdjustValue',
					'586B066FFED437A909EA3B8E665DE674': 'ICTAAdjustValue',
					'59565C0A77AF319AFD68C5F4166D9B30': 'Targetulsinr',
					'5AD2113932439FAB97323047197F1BAA': 'rootSequenceIndex1',
					'5F894D8A165C5E6FB079320FE78FDA9B': 'Hysteresis', // 待审
					'E488368D44546671C20F37361C54AD4E': 'B2IRATThreshold1', // 待审
					'6C5C6B04AC57D4C4DB940B5BC43E2FF0': 'Qhyst',
					'86B294B65A644E2BC5690C6E90819070': 'SNonIntraSearch',
					'A6BC02288CE97ACA2E11812A0ADD45CF': 'ULSchdAlgorithm',
					'F2C13B69F7E198C5848121020EB7D94E': 'DLSchdAlgorithm',
					'A7C688287C05443683B46486DC9A6677': 'PA',
					'B62AF1FB2D191F4D9BD638596951E07E': 'PRACHFreqOffset',
					'E4560E193F8A655DE89CB140D2F46834': 'WorkingMode',
					'EAF19D3B9E439CD83ACF3750032D0839': 'configurationIndex',
					'EC991C3DF5FEA75F3878F314E2AF0BFD': 'LinkKeepAlive',
					'FC7FAB34D54D0CCFAACA29DFFF0CE6EF': 'ZeroConfig',
				},
				neighType:'',
				operType:'',
				rowDataFreq:[],
				rowDataCell:[],
				smallCellCode:'',

				cellSeletions: [],
			}
		},
		watch: {
			lteForm: {
				handler: function(newVal, oldVal) {
					var vm = this,
						form = vm.$refs.lteForm;
					
					detectReboot(form, vm.rebootMap);
				},
				deep: true
			}
		},
		methods:{
			selectable(row, index) {
				// 不是輔站的，才可被選中
				if(row.neighborType == '2' || row.neighborType == '') {
					return false;
				}else {
					return true;
				}
			},
			cellSelectionChange(s) {
				this.cellSeletions = s || [];
			},
			batchSetPermanent() {
				var vm = this,
					indexList = vm.cellSeletions.map(item=>{
						return item.CellIndex;
					});

				vm.lteForm.NeighCellList.map(function(row){
					if(indexList.includes(row.CellIndex)) {
						if(row.neighborType == '2') {

						}else {
							row.neighborType = '2';
							row.operateType = 'edit';
						}
					}
				})
				vm.$refs.ctableCell.clearSelection();
			},
			batchSetBlockList() {
				var vm = this,
					indexList = vm.cellSeletions.map(item=>{
						return item.CellIndex;
					});

				vm.lteForm.NeighCellList.map(function(row){
					if(indexList.includes(row.CellIndex)) {
						if(row.neighborType == '2') {

						}else {
							row.neighborType = '3';
							row.operateType = 'edit';
						}
					}
				})
				vm.$refs.ctableCell.clearSelection();
			},
			setPermanent(row) {
				row.neighborType = '2';
				row.operateType = 'edit';
			},
			setBlockList(row) {
				row.neighborType = '3';
				row.operateType = 'edit';
			},
			initReboot(list, map) {
				var vm = this;
				
				list.map(function(item){
					if(item.reboot == '1') {
						var code = item.name,
							key = vm.casts[code];

						map[key] = true;
					}
				});
			},
            hasKey(key) {
                var vm = this,
                    has = false;

                vm.codeList.map(function(name){
                    if(vm.casts[name] == key) has = true;
                });

                return has;
            },
			init(code,id){
				var vm = this;
				vm.smallCellCode = code;
				vm.getParamNode(code,id);
			},
			getParamNode(code,id) {
                var vm = this,
                    codes = [],
                    url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
                    params = {
                        id: id,
                        smallCellCode: code
                    };

                axios.post(url, stringify(params)).then(function(res){
                    var data = res.data;
                    
                    if(data && Array.isArray(data)) {
                        data.map(function(item){
                            item.groups.map(function(group){
                                group.list.map(function(m){
                                	codes.push(m.name);
                                	if(m.type == 'list'){
										m.list.map(function(field){
											vm.codeTableList.push(field.name);
										});

                                		vm.initTable(m.url, m.label);
                                	}else{
                                   		// 执行赋值
                                        vm.setValue(m);
                                	}
                                });
								// 初始化重启项关系记录
								vm.initReboot(group.list, vm.rebootMap);
                            });
                        });
						vm.$nextTick(function(){
                            initForm(vm.$refs.lteForm);
                            detectReboot(vm.$refs.lteForm, vm.rebootMap);
							$('#setting_main').removeClass('loading');
                        });
                        vm.codeList = codes;
                    }
                });
            },
            setValue(item) {
            	var vm = this,
                code = item.name,
                value = item.value;

	            // indexs是否含有
	             var key = vm.casts[code];
	            if(key){
	            	vm.lteForm[key] = value;
	            }
	             
            },
            setListValue(item,type) {
            	var vm = this;

	            var obj = {};

				for(key in item) {
					var prop = vm.casts[key];

	            	obj[prop] = item[key];
				}

	            if(type == 'Neigh Freq List' || type == '邻频列表'){
	            	vm.lteForm.NeighFreqList.push(obj);
	            }else{
	            	vm.lteForm.NeighCellList.push(obj);
	            }
            },
            initTable(url, type){
            	var vm = this,
					params = {
						smallCellCode : vm.smallCellCode
					};
				
            	axios.post(url,stringify(params)).then(res=>{
            		var data = res.data;

            		if(data.rows){
            			data.rows.map((item, idx)=>{
            				vm.setListValue(item,type);
            			});

						if(type == 'Neigh Freq List' || type == '邻频列表') {
							vm.originNeigh.freq = data.rows;
						}else {
							vm.originNeigh.cell = data.rows;
						}
            		}

					if(vm.$refs.lteForm) initForm(vm.$refs.lteForm);
            	});
            },
            getNameByProp(prop) {
                var vm = this,
                    reg = /^\w*\.\d*\.\w*$/,
                    key = prop;
                
                if(reg.test(prop)) {
                    var mReg = /\.(\d*)\./,
                        sufReg = /\.(\w*)$/,
                        idx = prop.match(mReg)[1],
                        sufStr = prop.match(sufReg)[1];

                    vm.codeList.map(function(name){
                        var index = vm.indexs[name];
                        if(vm.casts[name] == sufStr && index == idx) {
                            key = name;
                        }
                    });
                }else {
                    (vm.codeList.concat(vm.codeTableList)).map(function(name){
                        if(vm.casts[name] == prop) {
                            key = name;
                        }
                    });
                }

                return key;
            },
            isNull(val){
                if(val==undefined || val == null || val =="") return true;
                else return false;
            },
			addFreq(){
				this.showNeigh = true;
				this.neighType = 'freq';
				this.operType = 'add';
				loadHTML(document.querySelector('#neighPanel'),{
                    url:'${ctx}/cell/quicksettings/goNeighFreqParamPage.action'
                });
			},
			editFreq(row){
				this.showNeigh = true;
				this.rowDataFreq = row;
				this.neighType = 'freq';
				this.operType = 'edit';
				loadHTML(document.querySelector('#neighPanel'),{
                    url:'${ctx}/cell/quicksettings/goNeighFreqParamPage.action'
                });
			},
			delFreq(row){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					var freqArr = vm.lteForm.NeighFreqList.map(function(item){
						return item.Index;
					})
					var index = freqArr.indexOf(row.Index);
					vm.lteForm.NeighFreqList.splice(index,1);
					// var length = vm.lteForm.NeighFreqList.length;
					// for(let i=0;i<length;i++){
					// 	vm.lteForm.NeighFreqList[i].Index = i+1;
					// }
					
					row.operateType = 'remove';
					vm.delRecord['NeighFreqList'].push(Object.assign({operateType: 'remove'},{Index: row.Index+''}));
				})
			},
			addCell(){
				this.showNeigh = true;
				this.neighType = 'cell';
				this.operType = 'add';
				document.querySelector('#neighPanel').innerHTML = '';
				loadHTML(document.querySelector('#neighPanel'),{
                    url:'${ctx}/cell/quicksettings/goNeighCellParamPage.action'
                });
			},
			editCell(row){
				this.showNeigh = true;
				this.rowDataCell = row;
				this.neighType = 'cell';
				this.operType = 'edit';
				document.querySelector('#neighPanel').innerHTML = '';
				loadHTML(document.querySelector('#neighPanel'),{
                    url:'${ctx}/cell/quicksettings/goNeighCellParamPage.action'
                });
			},
			delCell(row){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					var cellArr = vm.lteForm.NeighCellList.map(function(item){
						return item.CellIndex;
					})
					var index = cellArr.indexOf(row.CellIndex);
					vm.lteForm.NeighCellList.splice(index,1);
					// var length = vm.lteForm.NeighCellList.length;
					// for(let i=0;i<length;i++){
					// 	vm.lteForm.NeighCellList[i].CellIndex = i+1;
					// }
					
					row.operateType = 'remove';
					vm.delRecord['NeighCellList'].push(Object.assign({operateType: 'remove'},{CellIndex: row.CellIndex+''}));
				})
			},
			enableFmt(row,column,value,index){
				return (value+'').toString();
			},
			save(){
				var vm = this;
				if(vm.showNeigh){
					if(vm.neighType == 'freq'){
						eventBus.$emit('save-freq');
					}else{
						eventBus.$emit('save-cell');
					}
				}else{
					 var params = {},
	                    isChanged = isFormChanged(vm.$refs.lteForm);

	                if(!isChanged){
	                    showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
	                    return;
	                }

	                vm.$refs.lteForm.fields.map(function(field){
	                    var key = vm.getNameByProp(field.prop);

	                    if(Array.isArray(field.fieldValue)){
	                        var vList = field.fieldValue.map(function(item){return item}),
	                            oList = (field.reinitialValue||[]).map(function(item){return item}),
	                            val = JSON.stringify(vList.sort()),
	                            orVal = JSON.stringify(oList.sort());

	                        if(val != orVal) {
	                            params[key] = val;
								if(['NeighFreqList', 'NeighCellList'].includes(field.prop)) {
									var nList = [];
									vList.map(function(m){
										var obj = {},
											cellIndexName = vm.getNameByProp('CellIndex'),
											indexName = vm.getNameByProp('Index'),
											oldRow = vm.originNeigh.cell.filter(function(cRow){ //默认邻区
												return cRow[cellIndexName] == m.CellIndex;
											})[0];

											if('NeighFreqList' == field.prop) {
												oldRow = vm.originNeigh.freq.filter(function(cRow){
													return cRow[indexName] == m.Index;
												})[0];
											}
										
										if(m.operateType == 'edit') {
											if('NeighFreqList' == field.prop) {
												/*oldRow = vm.originNeigh.freq.filter(function(cRow){
													return cRow['0EE9F73309A62895413D4A6AA017E748'] == m.Index;
												})[0];*/
		
												obj[indexName] = oldRow[indexName];
											}else {
												obj[cellIndexName] = oldRow[cellIndexName];
											}
										}
										
										for(k in m) {
											var prop = vm.getNameByProp(k);

											if(m.operateType == 'edit') {
												if(m[k] != oldRow[prop]) {
													obj[prop] = m[k];
												}
											}else {
												if(['x2Flag','x2IP','x2Status'].includes(k)) {
													if(![null,undefined,''].includes(m[k].trim())) {
														obj[prop] = m[k];
													}
												}else {
													obj[prop] = m[k];
												}
											}
										}
										
										nList.push(obj);
									});

									params[key] = nList.filter(function(item){
										return ['edit','add'].includes(item.operateType);
									});
									
									if(vm.delRecord[field.prop].length) {
										vm.delRecord[field.prop].map(function(im){
											var obj = {};

											for(k in im) {
												var prop = vm.getNameByProp(k);

												obj[prop] = im[k];
											}
											
											params[key].push(obj);
										})
									}
								}
	                        };
	                    }else{
	                        if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
	                            
	                        }else if(field.fieldValue != field.reinitialValue) {
	                            params[key] = field.fieldValue;
	                        };
	                    }
	                });
	                
	                vm.$refs.lteForm.validate(function(valid){
	                    if(valid) {
							var isNeedReboot = detectReboot(vm.$refs.lteForm, vm.rebootMap);

							if(isNeedReboot) {
								var tipContent = [
										'<%=rb.getString("JiZhanChongQiTiShi")%>',
										'<br/><br/>',
										'<input id="reboot_confirm_status" type="checkbox" />',
										'<label for="reboot_confirm_status" style="font-size: 14px;color: #1DA3FC;cursor: pointer;"><%=rb.getString("SheZhiHouChongQi")%></label>'
									].join(" ");

								var msger = $.messager.confirm('<%=rb.getString("QueRen")%>', tipContent, function (r) {
									if(r) {
										/* 重启勾选判断 */
										var needReboot = false,
											rebootCkbox = $('#reboot_confirm_status',msger);
										if(rebootCkbox.length && rebootCkbox.prop('checked')){
											needReboot = true;
										}
										msger = null;
										
										var rowCode = vm.smallCellCode,
											url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
										
										$('#setting_main').addClass('loading');
										settingVue.submitDisabled = true;
										axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
											var data = res.data;
											if(data["success"]){
												vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});

												// 勾选重启，下发重启指令
												if(needReboot) {
													$.post("${ctx}/cell/cpeinfos/cellReboot.action", {cell_code: rowCode}, function (data) {
														if (!data["success"]) {
															showMsg('error_msg',data["message"]);
														}
													}, "json");
												}
												closeSettingPanel();
											}else{
												vm.$message.error(data["message"])
											}

											$('#setting_main').removeClass('loading');
											settingVue.submitDisabled = false;
										})
									}
								}).addClass("seriousConfirm");
							}else {
								var rowCode = vm.smallCellCode,
									url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
										
								$('#setting_main').addClass('loading');
								settingVue.submitDisabled = true;
								axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
									var data = res.data;
									if(data["success"]){
										vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
										closeSettingPanel();
									}else{
										vm.$message.error(data["message"])
									}

									$('#setting_main').removeClass('loading');
									settingVue.submitDisabled = false;
								})
							}
	                    }
	                });
				}
			},
			cancel(){
				var vm = this;
				if(isFormChanged(vm.$refs.lteForm)){//返回true为改变
					vm.$confirm("<%=rb.getString("QueDingLiKaiDangQianYeMian")%>",'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						closeSettingPanel();
					}).catch(() => {})
				}else{
					closeSettingPanel();
				}
			}
		},
		mounted(){
			eventBus.$off('tab-param').$on('tab-param',this.init);
			eventBus.$off('save-set').$on('save-set',this.save);
			eventBus.$off('cancel-set-tab').$on('cancel-set-tab',this.cancel);
		}
	})
</script>
<style>
	.el-form-item__label{
		margin-left:15px;
	}
	.unit-cls{
		display:inline-block;
		width:50px;
		height:24px;
		margin-left:-20px;
		line-height:24px;
		border:1px solid #DEDFE6;
		background:#F5F7FA;
		border-left:none;
		text-align:center;
		color:#000;
		border-radius:0px 4px 4px 0px;
		margin-right:20px;
	}
	.unit-item .el-input__inner{
		width:150px;
	}
</style>