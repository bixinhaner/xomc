<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<style>
	#batchConfigOperContent .title-text::after{
		display:none;
	}
	#batchConfigOperContent .el-form-item{
		display:inline-block;
		width: 33%;
		margin-bottom:18px;
	}
	
	#batchConfigOperContent .el-input__icon{
		line-height:28px;
	}
	#batchConfigOperContent .inputStar{
		position:absolute;
		left:15px;
		top:3px;
	}
	#batchConfigOperContent .selectStar{
		position:absolute;
		right:-30px;
		top:43px;
	}
	#batchConfigOperContent .el-tooltip__popper{
		padding:5px 10px;
	}
	#batchConfigOperContent .el-form-item__label{
		float:none;
	}
	#batchConfigOperContent .el-input__inner {
		min-height: 25px;
	}
	#batchConfigOperContent{
		padding:30px;
	}
	#batchConfigOperContent .plmnWarp .el-input__inner {
	    width: 200px !important;
	}
	#batchConfigOperContent .plmnWarp .el-input-group__append {
        border: 0;
        background: #FFFFFF;
    }
    #batchConfigOperContent .el-input-group__append{
        border-radius:0px;
        width:auto;
    }
    #batchConfigOperContent .mmeSelect .el-input,.mmeSelect .el-input__inner{
        width:100px !important;
    }
    #batchConfigOperContent .validate-item .el-input__inner{
        width:200px !important;
    }
    #batchConfigOperContent .validate-item .el-input-group__append{
        border:none;
        background:none;
    }
    #batchConfigOperContent .validate-item .el-form-item__error{
        display:none;
    }
    #batchConfigOperContent  .is-error .el-input-group__append,.is-error .item-tip{
        color:#FA5555;
    }
    #batchConfigOperContent .form-suffix,
    #batchConfigOperContent .form-suffix .text {
        line-height: 22px;
    }
</style>
<!-- #66668-10.2.1 ：BALBLQ:BLQ、 QA_436Q:436Q、 Intel_CR:4860-->
<div id='batchConfigOperContent'>
	<!-- 基础配置修改 -->
	<el-form :model='configForm' :rules='rules' ref='configForm' style='margin-left:20px;' v-if="activeName == 'batch'">
        <!--BAIBLQ、includes('QA_436Q')、includes('Intel_CR') 等可配置多组  PLMN , MME IP + PLMN;--><!-- 4860-Intel_CR_DC,辅小区可配置多个plmn-->
        <div v-if="((curPlatformType == 'BAIBLQ' || curPlatformType == 'MLQ' || curPlatformType.includes('QA_436Q') == true || curPlatformType.includes('Intel_CR') == true || curPlatformType.includes('MLN') == true) && isReadOnly == false) || (( curPlatformType == 'Intel_CR_DC' || curPlatformType == 'MLN_DC') && isReadOnly == true)">
            <div class='group-title'>
                <span class='title-icon'></span>
                <span class='title-text'>MME</span>
            </div>
            <div class='commonFlex'>
                <div style='width: 47%;'>
                    <el-form-item label="PLMN" style='margin-bottom:0px;position:relative;width: 100%;' :class="plmnCls" class='plmnWarp validate-item'>
                        <el-input v-model='plmnVal'>
                            <template slot="append">Range:5~6,No more than 6,Not repeat</template>
                        </el-input>
                        <span v-if="plmnGroup.length<6" @click='addPlmn' class='form-bt el-icon el-icon-plus' style='position:absolute;left:165px;top:-5px;'></span>
                    </el-form-item>
                    <div style='overflow:auto; width: 60%;'>
                        <el-form-item class='suffixItem' v-for='(domain,index) in plmnGroup' style='width:160px; margin-bottom: 10px;'>
                            <div class='form-suffix'>
                                <span class='text'>{{domain}}</span>
                                <span style='font-size:12px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click="removePlmn(domain,index)"></span>
                            </div>
                        </el-form-item>
                        <el-form-item prop="plmn_id" style='width: 100%'>
                            <el-input v-model="configForm.plmn_id" v-show=false></el-input>
                        </el-form-item>
                    </div>
                </div>
                <!--Intel_CR_DC && sn-2（后缀是-2）时： MME IP + PLMN 隐藏；所有隐藏字段，有值传值； -->
                <div v-if="(curPlatformType == 'Intel_CR_DC' || curPlatformType == 'MLN_DC') && isReadOnly == true"> </div>
                <div style='width: 52%;' v-else>
                    <el-form-item label="MME IP" style='margin-bottom:0px;width: 610px;' :class="mmeCls" class=''>
                        <el-input v-model='mmeVal'>
                            <template slot="append">PLMN</template>
                        </el-input>
                        <el-select v-if="(curPlatformType == 'Intel_CR_DC' || curPlatformType == 'MLN_DC') && isReadOnly == false" style='vertical-align:bottom;margin-left:-3px;' class='mmeSelect' v-model='mme_plmn'>
                            <el-option v-for="item in combPlmnGroup" :label="item" :value="item"></el-option>
                        </el-select>
                        <el-select v-else style='vertical-align:bottom;margin-left:-3px;' class='mmeSelect' v-model='mme_plmn'>
                            <el-option v-for="item in plmnGroup" :label="item" :value="item"></el-option>
                        </el-select>
                        <span v-if="mmeGroup.length<16" class='el-icon el-icon-plus' style='vertical-align:middle;margin-left:1px;' @click='addMME("mme_plmn")'></span>
                        <span class='item-tip'>No more than 16,Not repeat</span>
                    </el-form-item>
                    <div style='overflow:auto; width: 92%'>
                        <el-form-item class='suffixItem' v-for='(domain,index) in mmeGroup' style='width:260px;margin-bottom:8px; margin-right: 10px;'>
                            <div class='form-suffix' style='min-width:255px;'>
                                <span class='text' style='min-width:100px;border-right:1px solid #A0C4F9;height:22px;'>{{domain.mme}}</span>
                                <span class='text' style='min-width:100px;'>PLMN:<span>{{domain.plmn}}</span></span>
                                <span style='font-size:12px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click='removeMME(index)'></span>
                            </div>
                        </el-form-item>
                        <el-form-item prop="mmeStrNew" style='width: 100%'>
                            <el-input v-model="configForm.mmeStrNew" v-show=false></el-input>
                        </el-form-item>
                    </div>
                </div>
            </div>
        </div>
        <!--基本信息-->
		<div class='group-title' style='padding-top: 20px;'>
            <span class='title-icon'></span>
            <span class='title-text'><%=rb.getString("JiChuPeiZhi")%></span>
        </div>

        <!--Intel_CR_DC &&　SN-2：需置灰-->
        <!--QA_436Q_DC && SN-2: 需隐藏-->
        <div v-if="curPlatformType == 'QA_436Q_DC' && isReadOnly == true">
            <el-form-item label='<%=rb.getString("PinDian")%>' prop='frequency' v-if='curPlatformType != "Intel_CR_CA" && curPlatformType != "QA_436Q_CA" && curPlatformType != "MLN_CA"'>
                <el-tooltip effect='light' :content='tipFre' placement='bottom-end' :enterable=false>
                    <el-input v-model='configForm.frequency' @focus='changeToEarfcn' :disabled="isReadOnly == true && (curPlatformType == 'Intel_CR_DC' || curPlatformType == 'MLN_DC')"></el-input>
                </el-tooltip>
            </el-form-item>
            <el-form-item label='ECI (ECI=eNB_ID*256+Cell_ID)' prop='cell_identity' v-if='curPlatformType != "Intel_CR_CA" && curPlatformType != "QA_436Q_CA" && curPlatformType != "MLN_CA"'>
                <el-tooltip effect='light' :content='tipCell' placement='top-start' :enterable=false>
                    <el-input v-model='configForm.cell_identity'>
                        <span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
                    </el-input>
                </el-tooltip>
            </el-form-item>
            <el-form-item label='<%=rb.getString("PCI")%>' prop='phycellid' v-if='curPlatformType != "Intel_CR_CA" && curPlatformType != "QA_436Q_CA" && curPlatformType != "MLN_CA"'>
                <el-tooltip effect='light' :content='tipPci' placement='bottom-start' :enterable=false>
                    <el-input v-model='configForm.phycellid'>
                        <span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
                    </el-input>
                </el-tooltip>
            </el-form-item>
            <el-form-item label="<%=rb.getString("CPETxPower")%>" prop="txPower">
                <el-select v-model="configForm.txPower" filterable>
                    <el-option v-for="item in powerList" :key="item.value" :label="item.label" :value="item.value"></el-option>
                </el-select>
            </el-form-item>
        </div>
        <div v-else>
		    <el-form-item label='<%=rb.getString("ZhiChiPinDuan")%>' prop='bands_support'>
                <el-tooltip effect='light' :content='tipBand' placement='bottom-end' :enterable=false>
                    <el-input v-model='configForm.bands_support' :disabled="isReadOnly == true && (curPlatformType == 'Intel_CR_DC' || curPlatformType == 'MLN_DC')">
                        <span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
                    </el-input>
                </el-tooltip>
            </el-form-item>

            <el-form-item label='<%=rb.getString("DaiKuan")%>' prop='band_width'>
                <!--:disabled="isReadOnly == true && curPlatformType == 'Intel_CR_DC'"-->
                <el-select v-model='configForm.band_width'>
                    <el-option v-for='item in band_width_options' :key='item.value' :label='item.label' :value='item.value'></el-option>
                </el-select>
                <span class='operationDiv operation_getFocus selectStar'></span>
            </el-form-item>
            <el-form-item label='<%=rb.getString("PinDian")%>' prop='frequency' v-if='curPlatformType != "Intel_CR_CA" && curPlatformType != "QA_436Q_CA" && curPlatformType != "MLN_CA"'>
                <el-tooltip effect='light' :content='tipFre' placement='bottom-end' :enterable=false>
                    <!--:disabled="isReadOnly == true && curPlatformType == 'Intel_CR_DC'"-->
                    <el-input v-model='configForm.frequency' @focus='changeToEarfcn'></el-input>
                </el-tooltip>
            </el-form-item>
            <!--即插即用模块限制：curProductName != "NBIOT" && curProductName != "QAFA" && curProductName != "QAFB"'；选项中的值也不一致-->
            <el-form-item v-show="!isQAFB" label='<%=rb.getString("ZiZhenPeiBi")%>' prop='subframe_assignment'>
                <el-select v-model='configForm.subframe_assignment' :disabled="isReadOnly == true && (curPlatformType == 'Intel_CR_DC' || curPlatformType == 'MLN_DC')">
                    <el-option v-for='item in subframe_assignment_options' :key='item.value' :label='item.label' :value='item.value'></el-option>
                </el-select>
            </el-form-item>
            <!--即插即用模块限制：curProductName != "NBIOT" && curProductName != "QAFA" && curProductName != "QAFB"'-->
            <el-form-item v-show="!isQAFB" label='<%=rb.getString("TeShuZiZhenPeiBi")%>' prop='special_subframe_patterns'>
                <el-select v-model='configForm.special_subframe_patterns' :disabled="isReadOnly == true && (curPlatformType == 'Intel_CR_DC' || curPlatformType == 'MLN_DC')">
                    <el-option v-for='item in special_subframe_patterns_options' :key='item.value' :label='item.label' :value='item.value'></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label='<%=rb.getString("PLMN")%>' prop='plmn_id' v-if="curPlatformType !== 'BAIBLQ' && curPlatformType !== 'MLQ' && curPlatformType.includes('QA_436Q') == false && curPlatformType.includes('Intel_CR') == false && curPlatformType.includes('MLN') == false">
                <el-tooltip effect='light' :content='tipPlmn' placement='bottom-start' :enterable=false>
                    <el-input v-model='configForm.plmn_id'>
                        <span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
                    </el-input>
                </el-tooltip>
            </el-form-item>
            <el-form-item label='<%=rb.getString("TAC")%>' prop='tac'>
                <el-tooltip effect='light' :content='tipTac' placement='bottom-start' :enterable=false>
                    <el-input v-model='configForm.tac' :disabled="isReadOnly == true && (curPlatformType == 'Intel_CR_DC' || curPlatformType == 'MLN_DC')">
                        <span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
                    </el-input>
                </el-tooltip>
            </el-form-item>
            <el-form-item label="<%=rb.getString("CPETxPower")%>" prop="txPower">
                <el-select v-model="configForm.txPower" filterable>
                    <el-option v-for="item in powerList" :key="item.value" :label="item.label" :value="item.value"></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label='ECI (ECI=eNB_ID*256+Cell_ID)' prop='cell_identity' v-if='curPlatformType != "Intel_CR_CA" && curPlatformType != "QA_436Q_CA" && curPlatformType != "MLN_CA"'>
                <el-tooltip effect='light' :content='tipCell' placement='top-start' :enterable=false>
                    <el-input v-model='configForm.cell_identity'>
                        <span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
                    </el-input>
                </el-tooltip>
            </el-form-item>
            <el-form-item label='<%=rb.getString("PCI")%>' prop='phycellid' v-if='curPlatformType != "Intel_CR_CA" && curPlatformType != "QA_436Q_CA" && curPlatformType != "MLN_CA"'>
                <el-tooltip effect='light' :content='tipPci' placement='bottom-start' :enterable=false>
                    <el-input v-model='configForm.phycellid'>
                        <span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
                    </el-input>
                </el-tooltip>
            </el-form-item>
            <!--即插即用模块限制：curProductName != "NBIOT" -->
            <el-form-item v-show="!isQAFB" label='<%=rb.getString("GenXuLieSuoYin")%>' prop='root_sequence_index'>
                <el-tooltip effect='light' :content='tipRoot' placement='bottom-start' :enterable=false>
                    <el-input v-model='configForm.root_sequence_index'>
                        <span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
                    </el-input>
                </el-tooltip>
            </el-form-item>
            <!-- 不是 BAIBLQ、includes('QA_436Q')、includes('Intel_CR') 等可配置 MME IP
            v-if="curPlatformType !== 'BAIBLQ' && curPlatformType !== 'MLQ' && curPlatformType.includes('QA_436Q') == false && curPlatformType.includes('Intel_CR') == false" && curPlatformType.includes('MLN') == false"
            -->
            <div style='width: 33%;' v-if="curPlatformType !== 'BAIBLQ' && curPlatformType !== 'MLQ' && curPlatformType.includes('QA_436Q') == false && curPlatformType.includes('Intel_CR') == false && curPlatformType.includes('MLN') == false">
                <el-form-item label="MME IP" style='margin-bottom:0px;position:relative;width: 100%' class="validate-item plmnWarp" :class="mmeCls">
                    <el-input v-model='mmeVal'>
                        <template slot="append"><%=rb.getString("IPDiZhi")%></template>
                    </el-input>
                    <span class='el-icon el-icon-plus form-bt' style='position:absolute;left:165px;top:-5px;' @click='addMME("mme")'></span>
                </el-form-item>
                <div style='overflow:auto; width: 92%;'>
                    <el-form-item class='suffixItem' v-for='(domain,index) in mmeGroup' style='width:160px;'>
                        <div class='form-suffix'>
                            <span class='text' style='height:22px;'>{{domain}}</span>
                            <span style='font-size:12px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click='removeMME(index)'></span>
                        </div>
                    </el-form-item>
                    <el-form-item prop="mmeArr" style='width: 100%;'>
                        <el-input v-model="configForm.mmeArr" v-show=false></el-input>
                    </el-form-item>
                </div>
            </div>
        </div>
        <!--Intel_CR_CA || QA_436Q_CA 等类型，Earfcn、ECI、PCI可配置两组, 反之这三个字段只可配置一组-->
        <div v-if='curPlatformType == "Intel_CR_CA" || curPlatformType == "QA_436Q_CA" || curPlatformType == "MLN_CA"' class='commonFlex'>
            <div style='width: 33%;'>
                <el-form-item label='<%=rb.getString("PinDian")%>' style='margin-bottom:0px;position:relative;width: 100%' class="validate-item plmnWarp" :class="frequencyCls">
                    <el-input v-model='frequencyVal'>
                        <template slot="append">No more than 2,Not repeat</template>
                    </el-input>
                    <span v-if="frequencyGroup.length<2" class='el-icon el-icon-plus form-bt' style='position:absolute;left:165px;top:-5px;' @click='commonAddParams("frequency")'></span>
                </el-form-item>
                <el-form-item class='suffixItem' v-for='(domain,index) in frequencyGroup' style='width:160px;'>
                    <div class='form-suffix'>
                        <span class='text' style='height:22px;'>{{translateToFre(domain)}}</span>
                        <span style='font-size:12px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click='commonRemoveParams("frequency",index)'></span>
                    </div>
                </el-form-item>
                <el-form-item prop="frequency" style='width: 100%'>
                    <el-input v-model="configForm.frequency" v-show=false></el-input>
                </el-form-item>
            </div>

            <div style='width: 33%;'>
                <el-form-item label="ECI (ECI=eNB_ID*256+Cell_ID)" style='margin-bottom:0px;position:relative;width: 100%' class="validate-item plmnWarp" :class="eciCls">
                    <el-input v-model='eciVal'>
                        <template slot="append">No more than 2,Not repeat</template>
                    </el-input>
                    <span v-if="eciGroup.length<2" class='el-icon el-icon-plus form-bt' style='position:absolute;left:165px;top:-5px;' @click='commonAddParams("eci")'></span>
                </el-form-item>
                <div>
                    <el-form-item class='suffixItem' v-for='(domain,index) in eciGroup' style='width:160px;'>
                        <div class='form-suffix'>
                            <span class='text' style='height:22px;'>{{domain}}</span>
                            <span style='font-size:12px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click='commonRemoveParams("eci", index)'></span>
                        </div>
                    </el-form-item>
                    <el-form-item prop="cell_identity" style='width: 100%'>
                        <el-input v-model="configForm.cell_identity" v-show=false></el-input>
                    </el-form-item>
                </div>
            </div>
            <div style='width: 33%;'>
                <el-form-item label='<%=rb.getString("PCI")%>' style='margin-bottom:0px;position:relative; width: 100%' class="validate-item plmnWarp" :class="pciCls">
                    <el-input v-model='pciVal'>
                        <template slot="append">No more than 2,Not repeat</template>
                    </el-input>
                    <span v-if="pciGroup.length<2" class='el-icon el-icon-plus form-bt' style='position:absolute;left:165px;top:-5px;' @click='commonAddParams("pci")'></span>
                </el-form-item>
                <div>
                    <el-form-item class='suffixItem' v-for='(domain,index) in pciGroup' style='width:160px;'>
                        <div class='form-suffix'>
                            <span class='text' style='height:22px;'>{{domain}}</span>
                            <span style='font-size:12px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click='commonRemoveParams("pci", index)'></span>
                        </div>
                    </el-form-item>
                    <el-form-item prop="phycellid" style='width: 100%'>
                        <el-input v-model="configForm.phycellid" v-show=false></el-input>
                    </el-form-item>
                </div>
            </div>
        </div>
	</el-form>

	<!-- 邻区配置修改 -->
	<el-form :model='cellConfigForm' :rules='cellRules' ref='cellConfigForm' style='margin-left:20px;'  v-if="activeName == 'neighborCellConfig'">
        <div class='group-title'>
            <span class='title-icon'></span>
            <span class='title-text'><%=rb.getString("JiChuPeiZhi")%></span>
        </div>
        <el-form-item label='<%=rb.getString("XiaoQuBianHao")%>' prop="cellIndex" class='inputCommon validateItem'>
            <el-select v-model='cellConfigForm.cellIndex'>
                <el-option label='Cell 1' value='1'></el-option>
                <el-option label="Cell 2" value='2'></el-option>
            </el-select>
        </el-form-item>
		<el-form-item label='<%=rb.getString("PinDian")%>' prop='earfcn'>
			<el-tooltip effect='light' :content='tipFre' placement='bottom-end' :enterable=false>
				<el-input v-model='cellConfigForm.earfcn'></el-input>
			</el-tooltip>
		</el-form-item>
		<el-form-item label='<%=rb.getString("PCI")%>' prop='pci'>
			<el-tooltip effect='light' :content='tipPci' placement='bottom-start' :enterable=false>
				<el-input v-model='cellConfigForm.pci'>
					<span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
				</el-input>
			</el-tooltip>
		</el-form-item>
		<el-form-item label='<%=rb.getString("XiaoQuTeDingPianYiLiang")%>' prop='qOffset'>
			<el-select v-model='cellConfigForm.qOffset'>
				<el-option v-for='item in qOffset_options' :key='item.value' :label='item.text' :value='item.value'></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label='<%=rb.getString("XiaoQuDuLiPianYiLiang")%>' prop='cio'>
			<el-select v-model='cellConfigForm.cio'>			
				<el-option v-for='item in cio_options' :key='item.value' :label='item.text' :value='item.value'></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label='<%=rb.getString("TAC")%>' prop='tac'>
			<el-tooltip effect='light' :content='tipTac' placement='bottom-start' :enterable=false>
				<el-input v-model='cellConfigForm.tac'>
					<span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
				</el-input>
			</el-tooltip>
		</el-form-item>
		<el-form-item label='<%=rb.getString("PLMN")%>' prop='plmn'>
			<el-tooltip effect='light' :content='tipPlmn' placement='bottom-start' :enterable=false>
				<el-input v-model='cellConfigForm.plmn'>
					<span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
				</el-input>
			</el-tooltip>
		</el-form-item>
        
		<el-form-item label='ECI' prop='cellId'>
			<el-tooltip effect='light' :content='tipCell' placement='top-start' :enterable=false>
				<el-input v-model='cellConfigForm.cellId'>
					<span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
				</el-input>
			</el-tooltip>
		</el-form-item>
        <el-form-item label='eNodeB Type' prop='eNodeBType'>
			<el-select v-model='cellConfigForm.eNodeBType'>	
                <el-option v-for='item in eNodeBType_options' :key='item.value' :label='item.label' :value='item.value'></el-option>		
			</el-select>
		</el-form-item>
	</el-form>
	
	<!-- 临频配置修改 -->
	<el-form :model='freqConfigForm' :rules='freqRules' ref='freqConfigForm' style='margin-left:20px;' v-if="activeName == 'neighborFrequencyConfig'">	
		<div class='group-title' style='padding-top: 50px;'>
            <span class='title-icon'></span>
            <span class='title-text'><%=rb.getString("JiChuPeiZhi")%></span>
        </div>
		<el-form-item label='<%=rb.getString("PinDian")%>' prop='earfcn'>
			<el-tooltip effect='light' :content='tipFre' placement='bottom-end' :enterable=false>
				<el-input v-model='freqConfigForm.earfcn'></el-input>
			</el-tooltip>
		</el-form-item>
		
		<el-form-item label='Q-OffsetRange' prop='qOffsetRange'>
			<el-select v-model='freqConfigForm.qOffsetRange'>
				<el-option v-for='item in qOffsetFreq_options' :key='item.value' :label='item.text' :value='item.value'></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label='Q-RxLevMin' prop='qRxLevMin'>
			<el-tooltip effect='light' content='<%=rb.getString("FanWei")%>: -70 ~ -22' placement='bottom-start' :enterable=false>
				<el-input v-model='freqConfigForm.qRxLevMin'>
					<span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
				</el-input>
			</el-tooltip>
		</el-form-item>
		<el-form-item label='<%=rb.getString("XiaoQuZhongXuanYouXianJi")%>' prop='reselectionPriority'>
			<el-tooltip effect='light' content='<%=rb.getString("FanWei")%>: 0 ~ 7' placement='bottom-start' :enterable=false>
				<el-input v-model='freqConfigForm.reselectionPriority'>
					<span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
				</el-input>
			</el-tooltip>
		</el-form-item>
		<el-form-item label='<%=rb.getString("GaoChongXuanMenXian")%>' prop='reselectionThreshHigh'>
			<el-tooltip effect='light' content='<%=rb.getString("FanWei")%>: 0 ~ 31' placement='bottom-start' :enterable=false>
				<el-input v-model='freqConfigForm.reselectionThreshHigh'>
					<span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
				</el-input>
			</el-tooltip>
		</el-form-item>
		<el-form-item label='<%=rb.getString("DiChongXuanMenXian")%>' prop='reselectionThreshLow'>
			<el-tooltip effect='light' content='<%=rb.getString("FanWei")%>: 0 ~ 31' placement='bottom-start' :enterable=false>
				<el-input v-model='freqConfigForm.reselectionThreshLow'>
					<span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
				</el-input>
			</el-tooltip>
		</el-form-item>
		<el-form-item label='<%=rb.getString("UEZuiDaFaSongGongLv")%>' prop='pMax'>
			<el-tooltip effect='light' content='<%=rb.getString("FanWei")%>: -127 or -33 ~ 33' placement='top-start' :enterable=false>
				<el-input v-model='freqConfigForm.pMax'>
					<span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
				</el-input>
			</el-tooltip>
		</el-form-item>	
		<el-form-item label='<%=rb.getString("ChongXuanDingShiQi")%>' prop='tReselectionEutra'>
			<el-tooltip effect='light' content='<%=rb.getString("FanWei")%>: 0 ~ 7' placement='top-start' :enterable=false>
				<el-input v-model='freqConfigForm.tReselectionEutra'>
					<span slot='suffix' class='operationDiv operation_getFocus inputStar'></span>
				</el-input>
			</el-tooltip>
		</el-form-item>	
	</el-form>
</div>

<script>
var batchConfigOperContentVue = new Vue({
	el:'#batchConfigOperContent',
	data(){
		var vm = this;
		var validateInteger = (rule,value,callback) => {
			var reg = /^[0-9]*$/;
			if(value == ''){
				callback(this.tipBand)
			}else{
				if(reg.test(value) && value >= rule.min && value <= rule.max){
					callback()
				}else{
					callback(this.tipBand)
				}
			}
		}
		var validateFre = (rule,value,callback) => {
			var reg = /^[0-9]*$/;
			var message = '<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+rule.min+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+rule.max;
			//此条件下 只验证长度
            if(vm.curPlatformType == 'Intel_CR_CA' || vm.curPlatformType == 'QA_436Q_CA' || vm.curPlatformType == 'MLN_CA'){
                if(vm.singleOrMoreEarfcn == true){
                    callback()
                }else {
                    callback()
                }
            }else {
                if(value == ""){
                    callback();
                }else{
                    if(value.indexOf("(") == -1){
                        var showFlag = translateToFre(value);
                        if(showFlag == false){

                        }else{
                            this.configForm.frequency = showFlag;
                            this.earfcn = value;
                        }
                        if(reg.test(value) && value >= rule.min && value <= rule.max){
                            callback()
                        }else{
                            callback(message)
                        }
                    }else{
                        var index = value.indexOf('(')
                        value = value.substring(0,index);
                        this.earfcn = value;
                        if(reg.test(value) && value >= rule.min && value <= rule.max){
                            callback()
                        }else{
                            callback(message)
                        }
                    }
                }
            }
		}
		var validatePlmn = (rule,value,callback) => {
			var reg = /^[0-9]*$/, message = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+rule.min+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+rule.max;
            //此条件下 只验证长度
            
            //if(vm.curPlatformType == 'BAIBLQ' || vm.curPlatformType == 'MLQ' || vm.curPlatformType.includes('QA_436Q') == true || vm.curPlatformType.includes('Intel_CR') == true){
            //(curPlatformType.includes('Intel_CR') == false || curPlatformType == 'Intel_CR_DC' && isReadOnly == true)
            if(vm.curPlatformType == 'BAIBLQ' || vm.curPlatformType == 'MLQ' || vm.curPlatformType.includes('QA_436Q') == true || vm.curPlatformType.includes('Intel_CR') == true || vm.curPlatformType.includes('MLN') == true){
                if(vm.plmnGroup.length == 0){
                    callback(message)
                }else {
                    callback()
                }
            }else {
                if(value == ''){
                    callback(message)
                }else{
                    if(reg.test(value) && value >= rule.min && value <= rule.max && value.length >=5 && value.length <= 6){
                        callback()
                    }else{
                        callback(message)
                    }
                }
            }
		}
		var validateCellPlmn = (rule,value,callback) => {
            var reg = /^[0-9]*$/;
            var message = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+rule.min+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+rule.max;
            if(value == ''){
                callback(message)
            }else{
                if(reg.test(value) && value >= rule.min && value <= rule.max && value.length >=5 && value.length <= 6){
                    callback()
                }else{
                    callback(message)
                }
            }
        }
		var validateRange = (rule,value,callback) => {
			var reg = /^(\d+\.\.){0,1}(\d+)$/;
			var message = '<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+rule.min+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+rule.max;
			var showFlag = false;
			if(value == ""){
				showFlag = true;
			}else{
				if (!reg.test(value)) {
					showFlag = true;
				}else{
					if(value.indexOf("..")>-1){
						value= value.split("..");
						if(value[1]-value[0]<=0){
							showFlag = true;
						}
						if(rule.min-value[0]>0){
							showFlag = true;
						}
						if(rule.max-value[1]<0){
							showFlag = true;
						}
					}else{
						if(rule.min-value>0){
							showFlag = true;
						}
						if(rule.max-value<0){
							showFlag = true;
						}
					}
				}
			}
			if(showFlag){
				callback(message)
			}else{
				callback()
			}
		}

        var validateECIRange = (rule,value,callback) => {
            var reg = /^(\d+\.\.){0,1}(\d+)$/, showFlag = false;
            var message = '<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+rule.min+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+rule.max;

            if(value == ""){
                showFlag = true;
            }else{
                //两组 手动添加, 走添加的逻辑
                if(vm.singleOrMoreEci == true){

                }else{
                    //只添加一组
                    if (!reg.test(value)) {
                        showFlag = true;
                    }else{
                        if(value.indexOf("..")>-1){
                            value = value.split("..");
                            if(value[1]-value[0]<=0){
                                showFlag = true;
                            }
                            if(rule.min-value[0]>0){
                                showFlag = true;
                            }
                            if(rule.max-value[1]<0){
                                showFlag = true;
                            }
                        }else{
                            if(rule.min-value>0){
                                showFlag = true;
                            }
                            if(rule.max-value<0){
                                showFlag = true;
                            }
                        }
                    }
                }
            }
            if(showFlag){
                callback(message)
            }else{
                callback()
            }
        }
        var validatePCIRange = (rule,value,callback) => {
            var reg = /^(\d+\.\.){0,1}(\d+)$/, showFlag = false;
            var message = '<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+rule.min+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+rule.max;

            if(value == ""){
                showFlag = true;
            }else{
                //两组 手动添加, 走添加的逻辑
                if(vm.singleOrMoreEci == true){

                }else{
                    //只添加一组
                    if (!reg.test(value)) {
                        showFlag = true;
                    }else{
                        if(value.indexOf("..")>-1){
                            value = value.split("..");
                            if(value[1]-value[0]<=0){
                                showFlag = true;
                            }
                            if(rule.min-value[0]>0){
                                showFlag = true;
                            }
                            if(rule.max-value[1]<0){
                                showFlag = true;
                            }
                        }else{
                            if(rule.min-value>0){
                                showFlag = true;
                            }
                            if(rule.max-value<0){
                                showFlag = true;
                            }
                        }
                    }
                }
            }
            if(showFlag){
                callback(message)
            }else{
                callback()
            }
        }

		var isQAFB = '${platform}' == 'QAFB',
			ignoreValid = function(rule,value,callback){
				callback();
			};
		// earfcn
		var validateEarfcn = (rule,value,callback) => {
			var reg = /^[0-9]*$/;
			var message = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+rule.min+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+rule.max;
			if(value === "" || value === null || value === undefined){
				callback(message)
			}else if(reg.test(value) && value >= rule.min && value <= rule.max){
				callback()
			}else{
				callback(message)
			}
		},
		validateQOffset = (rule,value,callback) => {
			if(value === "" || value === null || value === undefined){
				callback('<%=rb.getString("QingXuanZe")%> <%=rb.getString("XiaoQuTeDingPianYiLiang")%>')
			}else{
				callback()
			}
		},
		validateRangeCio = (rule,value,callback) => {
			if(value === "" || value === null || value === undefined){
				callback('<%=rb.getString("QingXuanZe")%> <%=rb.getString("XiaoQuDuLiPianYiLiang")%>')
			}else{
				callback()
			}
		},
		validateQOffsetRange = (rule,value,callback) => {
			if(value === "" || value === null || value === undefined){
				callback('<%=rb.getString("QingXuanZe")%> Q-OffsetRange')
			}else{
				callback()
			}
		},
		validateFuRange = (rule,value,callback) => {
			var reg = /^\-\d+\.?\d*$/;
			if(value === "" || value === null || value === undefined){
				callback('<%=rb.getString("FanWei")%>: -70 ~ -22')
			}else if(!reg.test(value) || (value < -70 || value > -22)){
				callback('<%=rb.getString("FanWei")%>: -70 ~ -22')
			}else{
				callback()
			}			
		},
		validateRangeTwo = (rule,value,callback) => {
			var reg = /^[0-9]*$/;
			if(value === "" || value === null || value === undefined){
				callback('<%=rb.getString("FanWei")%>: 0 ~ 7 ')
			}else if(!reg.test(value) || (value < 0 || value > 7)){
				callback('<%=rb.getString("FanWei")%>: 0 ~ 7')
			}else{
				callback()
			}			
		},
		validateRangeThr = (rule,value,callback) => {
			var reg = /^[0-9]*$/;
			if(value === "" || value === null || value === undefined){
				callback('<%=rb.getString("FanWei")%>: 0 ~ 31')
			}else if(!reg.test(value) || (value < 0 || value > 31)){
				callback('<%=rb.getString("FanWei")%>: 0 ~ 31')
			}else{
				callback()
			}			
		},
		validateRangeFour = (rule,value,callback) => {
			var reg = /^-?[0-9]+.?[0-9]*$/;
			if(value === "" || value === null || value === undefined){
				callback('<%=rb.getString("FanWei")%>: -127 or -33 ~ 33')
			}else if(!reg.test(value) || ((value < -33 || value > 33) && value != -127)){
				callback('<%=rb.getString("FanWei")%>: -127 or -33 ~ 33')
			}else{
				callback()
			}			
		};
		
		return{
			tipBand:'<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>1<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>62',
			tipFre:'<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>65535',
			tipPlmn:'<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>00000<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>999999',
			tipTac:'<%=rb.getString("LiRu")%>:"12..34"<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>65535',
			tipCell:'<%=rb.getString("LiRu")%>:"12..34"<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>268435455',
			tipPci:'<%=rb.getString("LiRu")%>:"12..34"<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>503',
			tipRoot:'<%=rb.getString("LiRu")%>:"12..34"<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>837',
			//基础配置
			plmnGroup:[],
            combPlmnGroup: [], //4860 组合辅小区的plmn
			plmnVal:'',
			plmnCls:'',
            mmeType:'',
            mmeCls:'',
            mmeVal:'',
            mme_plmn:'',
            mmeGroup:[],
            //频点
            frequencyVal: '',
            frequencyCls:'',
            frequencyGroup: [],
            singleOrMoreEarfcn: false,
            singleOrMoreEci: false,
            singleOrMorePci: false,
            eciVal: '',
            eciCls:'',
            eciGroup: [],
            pciVal: '',
            pciCls:'',
            pciGroup: [],
			configForm:{
				bands_support:'${bands_support}',
				band_width:'${band_width}',
				frequency:this.changeToFrequency('${frequency}'),
				subframe_assignment:'${subframe_assignment}',
				special_subframe_patterns:'${special_subframe_patterns}',
				plmn_id:'${plmn_id}',
				tac:'${tac}',
                txPower: '',
				cell_identity:'${cell_identity}',
				phycellid:'${phycellid}',
				root_sequence_index:'${root_sequence_index}',
				mme:'',
				mmeArr:'${mme_ip}'.split(','),
				//plmnId: '',  //将换成 plmn_id
				mmeStrNew: '',
			},
			powerList:[],
			cellConfigForm:{				
				 earfcn:'',
				 pci:'',
				 qOffset:'',
				 cio:'',
				 tac:'',
				 plmn:'',
				 cellId:'',
                 cellIndex: '',
                 eNodeBType:''
			},
			//临频配置
			freqConfigForm:{				 
				 earfcn:'',
				 qOffsetRange:'',
				 qRxLevMin:'',
				 reselectionPriority:'',
				 reselectionThreshHigh:'',
				 reselectionThreshLow:'',
				 pMax:'',
				 tReselectionEutra:'',
			},
			band_width_options:[
				{
					label:'5MHz',
					value:'n25'
				},
				{
					label:'10MHz',
					value:'n50'
				},
				{
					label:'15MHz',
					value:'n75'
				},
				{
					label:'20MHz',
					value:'n100'
				}
			],
			subframe_assignment_options:[
				{label:'<%=rb.getString("QingXuanZe")%>',value:''},
				{label:'0(DL:UL = 1:3)',value:'0'},
				{label:'1(DL:UL = 2:2)',value:'1'},
				{label:'2(DL:UL = 3:1)',value:'2'},
				{label:'6(DL:UL = 3:5)',value:'6'}
			],					
			special_subframe_patterns_options:[
				{label:'<%=rb.getString("QingXuanZe")%>',value:''},
				{label:'5',value:'5'},
				{label:'7',value:'7'}
			],
			rules:{
				bands_support:[
					{min:1,max:62,validator:validateInteger,trigger:'change'}
				],
				band_width:[
					{required:true,message:'<%=rb.getString("BiTian")%>',trigger:'change'}
				],
				frequency:[
					{min:0,max:65535,validator:validateFre,trigger:'blur'}
				],
				plmn_id:[
					{min:'00000',max:'999999',validator:validatePlmn,trigger:'change'}
				],
				tac:[
					{min:0,max:65535,validator:validateRange,trigger:'change'}
				],
				cell_identity:[
					{min:0,max:268435455,validator: validateECIRange,trigger:'change'}
				],
				phycellid:[
					{min:0,max:503,validator: validatePCIRange,trigger:'change'}
				],
				root_sequence_index:[
					{min:0,max:837,validator: isQAFB?ignoreValid:validateRange,trigger:'change'}
				]
			},
			//邻区
			cellRules:{				
				earfcn:[
					{min:0,max:65535,validator:validateEarfcn,trigger:'change'}
				],
				pci:[
					{min:0,max:503,validator:validateRange,trigger:'change'}
				],
				tac:[
					{min:0,max:65535,validator:validateRange,trigger:'change'}
				],
				plmn:[
					{min:'00000',max:'999999',validator:validateCellPlmn,trigger:'change'}
				],
				cellId:[
					{min:0,max:268435455,validator:validateRange,trigger:'change'}
				],
				qOffset:[
					{validator:validateQOffset,trigger:'change'}
				],
				cio:[
					{validator:validateRangeCio,trigger:'change'}
				],
			},
			//临频
			freqRules:{
				earfcn:[
					{min:0,max:65535,validator:validateEarfcn,trigger:'change'}
				],
				qOffsetRange:[
					{validator:validateQOffsetRange,trigger:'change'}
				],
				qRxLevMin:[
					{validator:validateFuRange,trigger:'change'}
				],
				reselectionPriority:[
					{validator:validateRangeTwo,trigger:'change'}
				],
				reselectionThreshHigh:[
					{validator:validateRangeThr,trigger:'change'}
				],
				reselectionThreshLow:[
					{validator:validateRangeThr,trigger:'change'}
				],
				pMax:[
					{validator:validateRangeFour,trigger:'change'}
				],
				tReselectionEutra:[
					{validator:validateRangeTwo,trigger:'change'}
				],				
			},
			errorMsg:'<%=rb.getString("IPDiZhi")%>',
			existMsg:'<%=rb.getString("YiCunZai")%>',
			id:'',	
			isQAFB: '${platform}' == 'QAFB',
			activeName:'',
			cio_options:[],
			qOffset_options:[],
			qOffsetFreq_options:[],
			curPlatformType: '',
			isReadOnly: false,// Intel_CR_DC && sn-2 后缀：字段置灰不可编辑
            curSecondaryArr: [],
            eNodeBType_options: [
                {label:'Macro',value:'0'},
                {label:'Home',value:'1'}
            ]
		}
	},

	methods:{	
		modifyConfig(id,curTab,rowData){
			var vm = this,
				powerList = ['-20dBm','-10dBm','0dBm','1dBm','2dBm','3dBm','4dBm','5dBm','6dBm','7dBm','8dBm','9dBm','10dBm','11dBm','12dBm','13dBm','14dBm','15dBm','16dBm','17dBm','18dBm','19dBm','20dBm','21dBm','22dBm','23dBm','24dBm','25dBm','26dBm','27dBm','28dBm','29dBm','30dBm','31dBm','32dBm','33dBm','34dBm','35dBm','36dBm','37dBm','38dBm','39dBm','40dBm','41dBm','42dBm','43dBm','44dBm','45dBm','46dBm'],
                powerValList = [-20,-10,0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46];
			vm.id = id;
			vm.activeName = curTab;
			vm.powerList = powerList.map((item,index)=>{
                return {label:item,value:powerValList[index]}
            });

			if(curTab == 'batch'){
                vm.curPlatformType = rowData.platformType;

                if(rowData.serial_number.includes('-2')){
                    //不可编辑
                    vm.isReadOnly = true;
                }else{
                    vm.isReadOnly = false;                   
                }

                var curPlmnId= rowData.plmn_id,
                    curMMEIp= rowData.mme_ip,
                    curEarfcn = rowData.frequency,
                    curEci = rowData.cell_identity,
                    curPci = rowData.phycellid;
                //init
                if(vm.curPlatformType == "BAIBLQ" || vm.curPlatformType == "MLQ" || vm.curPlatformType.includes("QA_436Q") == true || vm.curPlatformType.includes("Intel_CR") == true || (vm.curPlatformType == "Intel_CR_DC" && vm.isReadOnly == true) || (vm.curPlatformType == "MLN_DC" && vm.isReadOnly == true)){
                    //plmn 4860 需组合主小区+辅小区的 plmn
                     //主小区 plmn  不变
                     if(curPlmnId){
                        vm.plmnGroup = curPlmnId.split(',');
                    }else{
                        vm.plmnGroup = [];
                    }

                    if((vm.curPlatformType == "Intel_CR_DC" || vm.curPlatformType == "MLN_DC") && vm.isReadOnly == false){
                        //将辅小区的查询的 PLMN 数据 添加到 mme ip + plmn 中， 左侧 plmn 数据则不变，只针对 4860 类型
                        axios.post('${ctx}/task/BatchConfiguration/getBatchConfigDeviceSecondCellPlmn.action',stringify({
                            serialNumber: rowData.serial_number
                        })).then(function(response){
                            var data = response.data;	

                            //组合主小区与辅小区数据
                            if(data && curPlmnId){
                                var minArr = curPlmnId.split(','),
                                    secondaryArr = data?data.toString().split(','):[];
                                    vm.curSecondaryArr = data?data.toString().split(','):[];
                                vm.combPlmnGroup = minArr.concat(secondaryArr.filter(function(v){
                                    return minArr.indexOf(v) === -1
                                }));
                            }else{
                                vm.combPlmnGroup = curPlmnId.split(',');
                            }                                                          
                        });
                    }
                   
                    //mme ip + plmn
                    if(curMMEIp){
                       curMMEIp.split(',').map(function(mmePlmn){
                            var mmeArr = mmePlmn.split(':');
                            vm.mmeGroup.push({mme: mmeArr[0], plmn: mmeArr[1]})
                        });
                    }else{
                         vm.mmeGroup = [];
                    }
                    //earfcn
                    if(curEarfcn){
                        vm.singleOrMoreEarfcn = true;
                        vm.frequencyGroup = curEarfcn.split(',');
                    }else{
                        vm.singleOrMoreEarfcn = false;
                        vm.frequencyGroup = [];
                    }
                    //eci
                    if(curEci){
                        vm.singleOrMoreEci = true;
                        vm.eciGroup = curEci.split(',');
                    }else{
                        vm.singleOrMoreEci = false;
                        vm.eciGroup = [];
                    }
                    //pci
                    if(curPci){
                        vm.singleOrMorePci = true;
                        vm.pciGroup = curPci.split(',');
                    }else{
                        vm.singleOrMorePci = false;
                        vm.pciGroup = [];
                    }
                }else{
                    if(curMMEIp){
                       curMMEIp.split(',').map(function(mmeIp){
                            vm.mmeGroup.push(mmeIp)
                       });
                    }else{
                         vm.mmeGroup = [];
                    }
                }

                if(rowData.txPower == '' || rowData.txPower == null){
                    vm.configForm.txPower = ''
                }else{
                    vm.powerList.map((item,index)=>{
                        if(item.value == rowData.txPower){
                            vm.configForm.txPower = item.value;
                        }
                    })
                }
			}else if(curTab == 'neighborCellConfig'){
			    //邻区默认回显数据
				axios.post('${ctx}/task/BatchConfiguration/getBatchNcellConfigFormatData.action',stringify({
					serialNumber: vm.serialNumber
				})).then(function(res){
					if(res.data) {
						res.data.filter(function(item){
							return item.field_name != undefined;
						}).map(function(item) {
							if(item.field_name == 'cio'){
								vm.cio_options = JSON.parse(item.data);
							}
							if(item.field_name == 'qOffset'){
								vm.qOffset_options = JSON.parse(item.data);
							}
						});
					}
				});
				
				if(rowData){
					var rowDataList = ['earfcn','pci','qOffset','cio','tac','plmn','cellId', 'cellIndex', 'eNodeBType'];
					rowDataList.map(function(prop){
						vm.cellConfigForm[prop]= rowData[prop];						
                        vm.cellConfigForm.cellIndex = rowData['cellIndex'] || '';
                        vm.cellConfigForm.eNodeBType = rowData['eNodeBType'] || '';
					});
				}
				
			}else if(curTab == 'neighborFrequencyConfig'){
				//临频默认回显数据
				// 获取邻频 Column
				axios.post('${ctx}/task/BatchConfiguration/getBatchNfreqConfigFormatData.action',stringify({
					serialNumber: vm.serialNumber
				})).then(function(res){	
					if(res.data) {
						res.data.filter(function(item){
							return item.field_name != undefined;
						}).map(function(item) {
							if(item.field_name == 'qOffSetFreq'){
								vm.qOffsetFreq_options = JSON.parse(item.data);
							}
						});
					}
				});
				if(rowData){
					var rowDataList = ['earfcn','qOffsetRange','qRxLevMin','reselectionPriority','reselectionThreshHigh','reselectionThreshLow','pMax','tReselectionEutra'];
					rowDataList.map(function(prop){
						vm.freqConfigForm[prop]= rowData[prop];						
					});
				}				
			}				
		},
        addPlmn(){
            var vm = this, value = vm.plmnVal,
                reg = /^(\d+\.\.){0,1}(\d+)$/,
                existed = false;

            vm.plmnGroup.map(function(item){
                if(item == value) existed = true;
            });

            if(existed || value == '' || !reg.test(value) || value.length < 5 || value.length > 6 || value == undefined || vm.plmnGroup.length > 5){
                vm.plmnCls = 'is-error';
            }else{
                vm.plmnCls = '';
                vm.plmnGroup.push(value);
                vm.plmnVal = '';
            }
        },
        removePlmn(domain,index){
            var vm = this;

            vm.plmnGroup.splice(index,1);
            if((vm.curPlatformType == "Intel_CR_DC" || vm.curPlatformType == "MLN_DC") && vm.isReadOnly == false){
                var curPlmnGroup =vm.combPlmnGroup.filter(function(item){
                    return item != domain
                })
                //合并数组curPlmnGroup  vm.curSecondaryArr，并去重
                vm.combPlmnGroup = vm.curSecondaryArr.concat(curPlmnGroup).filter(function(item,index,arr){
                    return arr.indexOf(item) === index;
                })
            }            
            vm.plmnCls = '';
        },
        isMMEExisted(type) {
            var vm = this,
                mme = vm.mmeVal,
                plmn = vm.mme_plmn,
                existed = false;

            vm.mmeGroup.map(function(item){
                if(type == 'mme'){
                    if(item == mme) existed = true;
                }else {
                    if(item.mme == mme && item.plmn == plmn) existed = true;
                }
            })

            return existed;
        },
        addMME(type){
            var vm = this, value = vm.mmeVal;

            vm.mmeType = type;

            if(!isValidIP(value) || vm.isMMEExisted(type)){
                vm.mmeCls = 'is-error';
            }else{
                vm.mmeCls = '';
                if(type == 'mme'){
                    vm.mmeGroup.push(vm.mmeVal);
                }else{
                    vm.mmeGroup.push({mme: vm.mmeVal, plmn: vm.mme_plmn})
                }
                vm.mmeVal = '';
            }
        },
        removeMME(index){
            var vm = this;

            vm.mmeVal = '';
            vm.mme_plmn = '';
            vm.mmeGroup.splice(index,1);
        },
        isNull(val){
            if(val==undefined || val == null || val =="") return true;
            else return false;
        },


        //配置两组锁频
        commonAddParams(type){
            var vm = this,  value = '',  existed = false;

            if(type == 'frequency'){
                vm.singleOrMoreEarfcn = true;
                var reg = /^[0-9]*$/;

                value = vm.frequencyVal;
                vm.frequencyGroup.map(function(item){
                    if(item == value) existed = true;
                });
                if(existed || value == '' || !reg.test(value) || value < 0 || value > 65535 || value == undefined || vm.frequencyGroup.length > 2){
                    vm.frequencyCls = 'is-error';
                }else{
                    vm.frequencyCls = '';
                    vm.frequencyGroup.push(value);
                    vm.frequencyVal = '';
                }
            }else if(type == 'eci'){
                vm.singleOrMoreEci = true;
                value = vm.eciVal;
                var reg = /^(\d+\.\.){0,1}(\d+)$/, showFlag = false;

                vm.eciGroup.map(function(item){
                    if(item == value) existed = true;
                });

                if(value == "" || value == undefined || value == null){
                    showFlag = true;
                }else{
                    if (!reg.test(value)) {
                        showFlag = true;
                    }else{
                        if(value.indexOf("..") > -1){
                            var curValue = value.split("..");
                            if(curValue[1] - curValue[0] <= 0){
                                showFlag = true;
                            }
                            if(0 - curValue[0] > 0){
                                showFlag = true;
                            }
                            if(268435455 - curValue[1] < 0){
                                showFlag = true;
                            }
                        }else{
                            if(0 - value > 0){
                                showFlag = true;
                            }
                            if(268435455 - value < 0){
                                showFlag = true;
                            }
                        }
                    }
                }
                if(showFlag || existed || vm.eciGroup.length > 2){
                     vm.eciCls = 'is-error';
                }else{
                    vm.eciCls = '';
                    vm.eciGroup.push(value);
                    vm.eciVal = '';
                }
            }else if(type == 'pci'){
                var reg = /^(\d+\.\.){0,1}(\d+)$/, showFlag = false;

                vm.singleOrMorePci = true;
                value = vm.pciVal;
                vm.pciGroup.map(function(item){
                    if(item == value) existed = true;
                });

                if(value == "" || value == undefined || value == null){
                    showFlag = true;
                }else{
                    if (!reg.test(value)) {
                        showFlag = true;
                    }else{
                        if(value.indexOf("..") > -1){
                            var curValue = value.split("..");
                            if(curValue[1] - curValue[0] <= 0){
                                showFlag = true;
                            }
                            if(0 - curValue[0] > 0){
                                showFlag = true;
                            }
                            if(503 - curValue[1] < 0){
                                showFlag = true;
                            }
                        }else{
                            if(0 - value > 0){
                                showFlag = true;
                            }
                            if(503 - value < 0){
                                showFlag = true;
                            }
                        }
                    }
                }
                if(showFlag || existed || vm.pciGroup.length > 2){
                     vm.pciCls = 'is-error';
                }else{
                    vm.pciCls = '';
                    vm.pciGroup.push(value);
                    vm.pciVal = '';
                }
            }
        },

        commonRemoveParams(type, index){
            var vm = this;

            if(type == 'frequency'){
                vm.frequencyVal = '';
                vm.frequencyGroup.splice(index,1);
            }else if(type == 'eci'){
                vm.eciVal = '';
                vm.eciGroup.splice(index,1);
            }else if(type == 'pci'){
                vm.pciVal = '';
                vm.pciGroup.splice(index,1);
            }
        },
		submit(){
			var vm = this, params = {};
			//邻区提交
			if(vm.activeName == 'neighborCellConfig'){	
				vm.$refs.cellConfigForm.validate((valid) => {
					if(valid){
						vm.cellConfigForm.id = vm.id;
						axios.post('${ctx}/task/BatchConfiguration/updateBatchNeighbourCell.action',stringify(vm.cellConfigForm)).then(function(response){
							var data = response.data;
		    				if(data["success"]){
		    					vm.$message({
		    						message:'<%=rb.getString("XiuGai")%><%=rb.getString("ChengGong")%>',
		    						type:'success',
		    					})
                                eventBus.$emit('cancel-slide')
		    				}else{
		    					vm.$message.error(data["message"])
		    				}
						})
					}
				})
			}else if(vm.activeName == 'neighborFrequencyConfig'){
				//临频提交
				vm.$refs.freqConfigForm.validate((valid) => {
					if(valid){
						vm.freqConfigForm.id = vm.id;

						axios.post('${ctx}/task/BatchConfiguration/updateBatchNeighbourFreq.action',stringify(vm.freqConfigForm)).then(function(response){
							var data = response.data;

		    				if(data["success"]){
		    					vm.$message({
		    						message:'<%=rb.getString("XiuGai")%><%=rb.getString("ChengGong")%>',
		    						type:'success',
		    					})
                                eventBus.$emit('cancel-slide')
		    				}else{
		    					vm.$message.error(data["message"])
		    				}
						})
					}
				})
			}else{
				//基础配置的修改
				var mmeIps = '';

				vm.$refs.configForm.validate((valid) => {
					if(valid){
					    if(vm.curPlatformType == 'BAIBLQ' || vm.curPlatformType == 'MLQ' || vm.curPlatformType.includes('Intel_CR') == true || vm.curPlatformType.includes('QA_436Q') == true || vm.curPlatformType.includes('MLN') == true){
					        mmeIps = vm.configForm.mmeStrNew;
					    }else {
					        mmeIps = vm.configForm.mmeArr;
					    }
					    //Intel_CR_CA || QA_436Q_CA 等类型，Earfcn、ECI、PCI可配置两组, 反之这三个字段只可配置一组
					    if(vm.curPlatformType == 'Intel_CR_CA' || vm.curPlatformType == 'QA_436Q_CA' || vm.curPlatformType == 'MLN_CA'){
                            vm.earfcn = vm.configForm.frequency;
                        }

                        if(vm.earfcn == '' || vm.earfcn == null || vm.earfcn == undefined){
                            vm.earfcn = '';
                        }
                        
                        //Intel_CR_DC && sn-2 后缀：frequency、cell_identity、phycellid、txPower 字段
                        if(vm.curPlatformType == 'QA_436Q_DC' && vm.isReadOnly == true){
                            params.frequency = vm.earfcn;
                            params.cell_identity = vm.configForm.cell_identity;
                            params.phycellid = vm.configForm.phycellid;
                            params.txPower = vm.configForm.txPower;
                            params.id = vm.id;
                        }else{
                            params = {
                                bands_support : vm.configForm.bands_support,
                                band_width : vm.configForm.band_width,
                                frequency : vm.earfcn,
                                subframe_assignment : vm.configForm.subframe_assignment,
                                special_subframe_patterns : vm.configForm.special_subframe_patterns,
                                plmn_id : vm.configForm.plmn_id,
                                tac : vm.configForm.tac,
                                txPower: vm.configForm.txPower,
                                cell_identity : vm.configForm.cell_identity,
                                phycellid : vm.configForm.phycellid,
                                root_sequence_index : vm.configForm.root_sequence_index,
                                mme_ip : mmeIps,
                                id : vm.id,
                            }
                        }
                        
						axios.post('${ctx}/task/BatchConfiguration/updateBatchConfiguration.action',stringify(params)).then(function(response){
							var data = response.data;

		    				if(data["success"]){
		    					vm.$message({
		    						message:'<%=rb.getString("XiuGai")%><%=rb.getString("ChengGong")%>',
		    						type:'success',
		    					})
                                eventBus.$emit('cancel-slide')
		    				}else{
		    					vm.$message.error(data["message"])
		    				}
						})
					}
				})
			}									
		},
		
		cancel(){
			var vm = this, curTable;

			if(vm.activeName == 'neighborCellConfig'){	
				curTable = vm.$refs.cellConfigForm;
			}else if(vm.activeName == 'neighborFrequencyConfig'){
				curTable = vm.$refs.freqConfigForm;
			}else{
				curTable = vm.$refs.configForm;
			}
			if(isFormChanged(curTable)){
				vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>',QueRen,{
					customClass:'warningConfirm',
		    		confirmButtonText:'<%=rb.getString("QueDing")%>',
		    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
		    		type:'warning',
		    		closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('cancel-modify')
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('cancel-modify')
			}
		},
		
		changeToFrequency(val){
			if(val == ""){
				return ""
			}else{
				var showFlag = translateToFre(val);
				if(showFlag == false){
					
				}else{
					 return showFlag;
				}
			}
		},
		changeToEarfcn(){	
            var vm = this;

			var val = vm.configForm.frequency;

			if(val == ""){
				
			}else{
				var showFlag = val.indexOf("(");
				if(showFlag == -1){
					
				}else{
					vm.configForm.frequency = val.substring(0,showFlag);
				}
			}
		}
	},
	watch:{
        plmnGroup(){
            var vm = this, minArr = vm.plmnGroup, secondaryArr = vm.combPlmnGroup;

            vm.configForm.plmn_id = vm.plmnGroup.toString();

            if((vm.curPlatformType == 'Intel_CR_DC' || vm.curPlatformType == 'MLN_DC') && vm.isReadOnly == false){
                vm.combPlmnGroup = minArr.concat(secondaryArr.filter(function(v){
                    return minArr.indexOf(v) === -1
                })).sort();
            } 
        },
        mmeGroup(){
            var vm = this;

            if(vm.mmeType == 'mme'){
                vm.configForm.mmeArr = (vm.mmeGroup || []).join(',');
            }else{
                vm.configForm.mmeStrNew = (vm.mmeGroup || []).map(function(item){
                    if(item.mme == '' || item.mme == null || item.mme == undefined){
                        item.mme = '';
                    }

                    if(item.plmn == '' || item.plmn == null || item.plmn == undefined){
                        item.plmn = '';
                    }
                    return item.mme + ':' + item.plmn
                }).join(',')
            }
        },
        frequencyGroup(){
            var vm = this;

            vm.configForm.frequency = vm.frequencyGroup.toString();
        },
        eciGroup(){
            var vm = this;

            vm.configForm.cell_identity = vm.eciGroup.toString();
        },
        pciGroup(){
            var vm = this;

            vm.configForm.phycellid = vm.pciGroup.toString();
        }
    },
    mounted(){
        eventBus.$off('modify-config').$on('modify-config',this.modifyConfig)
        eventBus.$off('save-config').$on('save-config',this.submit)
        eventBus.$off('cancel-config').$on('cancel-config',this.cancel)
    },
})
</script>