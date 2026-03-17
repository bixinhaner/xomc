<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#cpeLteOrNrPage{
	height: 100%;
	width: 100%;
}
#cpeLteOrNrPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#cpeLteOrNrPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#cpeLteOrNrPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#cpeLteOrNrPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#cpeLteOrNrPage .el-form-item{
	margin-bottom: 20px;
}
#cpeLteOrNrPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
#cpeLteOrNrPage .paramsItemBoxCls{
	display: flex;
    flex-direction: column;
	margin-bottom: 20px;
	min-width: 740px;
}
#cpeLteOrNrPage .paramsItemBoxCls .el-form-item{
	margin-bottom: 0px;
}
#cpeLteOrNrPage .paramsItemLabelCls{
	width: 80%;
    display: flex;
    align-items: center;
    margin-bottom: 5px;
}
#cpeLteOrNrPage .exportBtnBoxCls{
	height: 28px;
	width: 130px;
	margin-left: 140px;
	border : 1px solid #E9E9E9;
	border-radius: 4px;
	font-size: 12px;
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	margin-bottom: 20px;
}
#cpeLteOrNrPage .exportTemplateBoxCls{
	display: flex;
	align-items: center;
}
#cpeLteOrNrPage .exportTemplateTipCls{
	color: rgba(0,0,0,0.32);
	margin-right: 20px;
}

#cpeLteOrNrPage .validate-item .el-input__inner{
	width:200px;
}
#cpeLteOrNrPage .validate-item .el-input-group__append{
	border:none;
	background:none;
	padding: 0px 10px;
}
#cpeLteOrNrPage .validate-item .el-form-item__error{
	display:none;
}
#cpeLteOrNrPage .is-error .el-input-group__append{
	color:#FA5555;
}
#cpeLteOrNrPage .notSupportTipBoxCls{
	color: rgba(0,0,0,0.32);
	font-size: 14px;
	margin-left: 20px;
}
#cpeLteOrNrPage .itemMainBoxCls .el-collapse-item{
    position:relative;
    border-bottom:1px solid transparent;
}
#cpeLteOrNrPage .itemMainBoxCls  .el-collapse-item__wrap{
    border-bottom:1px solid transparent;
    padding-left: 0px;
}
#cpeLteOrNrPage .paramsItemLabelCls .allowMoreInputAddBtnCls{
    height: 26px;
    width: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    color:var(--main-color);
    border: 1px solid var(--main-color);
    background:rgba(var(--main-color-rgba1),0.1);
    border-radius: 4px;
    box-sizing: border-box;
    margin-left: 10px;
    cursor: pointer;
}
#cpeLteOrNrPage .paramsItemLabelCls .allowMoreInputAddTipCls{
    margin-left: 10px;
}
#cpeLteOrNrPage .paramsItemLabelCls .allowMoreInputAddBtnCls .el-icon::before{
    font-size: 16px;
    color:var(--main-color);
}
#cpeLteOrNrPage .paramsItemLabelCls .allowMoreInputAddBtnCls span:nth-child(2){
    margin: 0px 3px;
}
#cpeLteOrNrPage .allowMoreTableParamsItemCls{
    height: 34px;
    width: 100%;
    border: 1px solid #D5DCEC;
    border-radius: 4px;
    display: flex;
    align-items: center;
    position: relative;
    margin-bottom: 5px;
}
#cpeLteOrNrPage .allowMoreTableParamsItemCls .el-icon-close{
    position: absolute;
}
#cpeLteOrNrPage .allowMoreTableParamsItemCls .el-icon-close::before{
    color:#7A7992;
}
#cpeLteOrNrPage .allowMoreTableParamsItemCls:hover {
    background:rgba(var(--main-color-rgba1),0.1);
}
#cpeLteOrNrPage .allowMoreTableParamsItemCls:hover .el-icon-close::before{
    color:var(--main-color);
}
#cpeLteOrNrPage .add5GCpePciLockBoxCls{
    width: 100%;
    border: 1px solid #D5DCEC;
    border-radius: 4px;
    background: #F6F7FB;
    padding: 20px;
    box-sizing: border-box;
    margin-bottom: 20px;
}
#cpeLteOrNrPage .add5GCpePciLockBoxCls .el-form{
    display: flex;
    flex-wrap: wrap;
}
#cpeLteOrNrPage .add5GCpePciLockBoxCls .el-form-item{
    margin-bottom: 20px;
}
/** 可删除 */ 
#cpeLteOrNrPage .el-collapse-item__header,.cpeConfigAddDialog .el-collapse-item__header{
    border-bottom:1px solid #fff;
}
#cpeLteOrNrPage .el-collapse-item__arrow,.cpeConfigAddDialog .el-collapse-item__arrow{
    position:absolute;
    left:20px;
    top:0px;
}
#cpeLteOrNrPage .el-collapse-item,.cpeConfigAddDialog .el-collapse-item{
    position:relative;
    border-bottom:1px solid #E9E9E9;
}
#cpeLteOrNrPage .el-collapse-item__content,.cpeConfigAddDialog .el-collapse-item__content{
    margin: 0px 40px;
    padding-bottom: unset;
    position: relative;
}
#cpeLteOrNrPage .el-collapse-item__header .el-icon-arrow-right,.cpeConfigAddDialog .el-collapse-item__header .el-icon-arrow-right{
    font-size:16px;
}
#cpeLteOrNrPage .el-collapse-item__header .el-icon-arrow-right:before,.cpeConfigAddDialog .el-collapse-item__header .el-icon-arrow-right:before{
    content:"\e639";
    color:#BBB;
}
#cpeLteOrNrPage .el-collapse-item__header .is-active.el-icon-arrow-right:before,.cpeConfigAddDialog .el-collapse-item__header .is-active.el-icon-arrow-right:before{
    content:"\e638";
    color:#BBB;
}
#cpeLteOrNrPage .el-collapse-item__arrow.is-active,.cpeConfigAddDialog .el-collapse-item__arrow.is-active{
    transform:rotate(0deg);
}
#cpeLteOrNrPage .el-collapse,.cpeConfigAddDialog .el-collapse{
    border-top:1px solid #fff;
    border-bottom:1px solid #fff;
}
.cpeConfigAddDialog .el-collapse{
    width: 100%;
}
#cpeLteOrNrPage .el-collapse-item__wrap,.cpeConfigAddDialog .el-collapse-item__wrap{
    border-bottom:1px solid #fff;
    padding-left: 0px;
}
#cpeLteOrNrPage .el-collapse-item__header,.cpeConfigAddDialog .el-collapse-item__header{
    width:100%;
    position: relative;
}

#cpeLteOrNrPage  .allowMoreInputBoxCls{
    position: relative;
    flex: 1;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputHeadCls{
    margin-bottom: 5px;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTitleCls{
    font-size: 14px;
    color: rgba(0, 0, 0, 0.8);
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTipsCls{
    font-size: 14px;
    color: rgba(0, 0, 0, 0.32);
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputContentCls{
    border: 1px solid #DFE2EE;
    width: 80%;
    min-width: 600px;
    min-height: 78px;
    padding: 10px;
    border-radius: 4px;
    box-sizing: border-box;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputFieldCls{
    display: flex;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputFieldCls .el-input{
    width: 240px;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls{
    height: 26px;
    width: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    color:var(--main-color);
    border: 1px solid var(--main-color);
    background:rgba(var(--main-color-rgba1),0.1);
    border-radius: 4px;
    box-sizing: border-box;
    margin-left: 10px;
    cursor: pointer;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddTipCls{
    margin-left: 10px;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls .el-icon::before{
    font-size: 16px;
    color:var(--main-color);
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls span:nth-child(2){
    margin: 0px 3px;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputParamsCls{
    display: flex;
    flex-wrap: wrap;
    width: 100%;
    margin-top: 10px;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls{
    height: 26px;
    display: flex;
    align-items: center;
    border: 1px solid #DFE2EE;
    border-radius: 4px;
    box-sizing: border-box;
    padding: 0px 10px;
    margin-right: 10px;
    margin-bottom: 5px;
    background: #F8F8FD;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls span:nth-child(1){
    display: inline-block;
    max-width: 520px;
    overflow: hidden;
    text-overflow: ellipsis;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls span:nth-child(2){
    margin-left: 10px;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close{
    font-size: unset;
    position: unset;
    top: unset;
    right: unset;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close::before{
    font-size: 12px;
    color: #7A7992;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputFootCls{
    height: 18px;
}
#cpeLteOrNrPage .allowMoreInputBoxCls .allowMoreInputFootCls .inputErrorBoxCls{
    color:red;
    font-size:10px;
}
</style>
<div id="cpeLteOrNrPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxCenter">
            <el-collapse v-model="activeCollapse">
                <el-collapse-item name="4GCPE" v-show="cpeModelType == '4G'">
                    <template slot='title'>
                        <p style="display:inline-block;margin-left:40px;">
                            <span style="font-size:14px;font-weight:bold">Frequency Lock</span>
                        </p>
                    </template>
                    <div class="rightContentCls">
                        <el-form :model='ruleForm4G' ref="ruleForm4G" :rules="rules4G" label-position="top">
                            <div class="paramsItemBoxCls">
                                <div class="paramsItemLabelCls"><%=rb.getString("SaoMiaoFangShi")%></div>
                                <el-form-item prop='scanMode' style="width:80%;min-width:400px;" label="">
                                    <el-radio-group v-model="ruleForm4G.scanMode" >
                                        <el-radio label="fullband" border size="small">Full Band</el-radio>
                                        <el-radio label="freqpreferred" border size="small">Band/Frequency Preferred</el-radio>
                                        <el-radio label="pcilock" border size="small">PCI lock</el-radio>
                                        <el-radio label="pcionlylock" border size="small">PCI Only Lock</el-radio>
                                    </el-radio-group>
                                </el-form-item>
                            </div>
                            <div class="paramsItemBoxCls" style="margin-bottom: 5px;" v-show="ruleForm4G.scanMode == 'freqpreferred'">
                                <div class="paramsItemLabelCls">Earfcn</div>
                                <div class="allowMoreInputBoxCls">
                                    <div class="allowMoreInputContentCls">
                                        <div class="allowMoreInputFieldCls">
                                            <el-input v-model="ruleForm4G.earfcn4G"></el-input>
                                            <div class="allowMoreInputAddBtnCls" @click="earfcn4GAdd">
                                                <span class="el-icon el-icon-plus"></span>
                                                <span>Add</span>
                                            </div>
                                            <div class="allowMoreInputAddTipCls">
                                                <span style="color:red" v-if="earfcnErrorMessage4G">{{earfcnErrorMessage4G}}</span>
                                                <span style="color:rgba(0,0,0,0.32)" v-if="!earfcnErrorMessage4G"><%=rb.getString("FanWei")%>：0~65535,Integer</span>
                                            </div>
                                        </div>
                                        <div class="allowMoreInputParamsCls">
                                            <div v-for="item in ruleForm4G.earfcn4GList" class="allowMoreInputParamsItemCls">
                                                <span>{{item}}</span>
                                                <span class="el-icon el-icon-close" @click="earfcnList4GDel(item)"></span>
                                            </div>
                                        </div>
                                        <el-form-item prop='earfcn4GList' style="display:none;" label="" label-width="0px">
                                            <el-input v-model='ruleForm4G.earfcn4GList'></el-input>
                                        </el-form-item>
                                    </div>
                                </div>
                            </div>
                            <div class="paramsItemBoxCls" style="margin-bottom: 5px;" v-show="ruleForm4G.scanMode == 'pcilock'">
                                <div class="paramsItemLabelCls">Earfcn : PCI</div>
                                <div class="allowMoreInputBoxCls">
                                    <div class="allowMoreInputContentCls">
                                        <div class="allowMoreInputFieldCls">
                                            <el-input style="width: 120px;" v-model="ruleForm4G.pciLockEarfcn4G"></el-input>
                                            <el-input style="width: 120px;" v-model="ruleForm4G.pciLockPci4G"></el-input>
                                            <div class="allowMoreInputAddBtnCls" @click="earfcnOrPci4GAdd">
                                                <span class="el-icon el-icon-plus"></span>
                                                <span>Add</span>
                                            </div>
                                            <div class="allowMoreInputAddTipCls">
                                                <span style="color:red" v-if="earfcnOrPciErrorMessage4G">{{earfcnOrPciErrorMessage4G}}</span>
                                                <span style="color:rgba(0,0,0,0.32)" v-if="!earfcnOrPciErrorMessage4G">Earfcn <%=rb.getString("FanWei")%>：0 ~65535, PCI <%=rb.getString("FanWei")%>：0~503</span>
                                            </div>
                                        </div>
                                        <div class="allowMoreInputParamsCls">
                                            <div v-for="item in ruleForm4G.pciLockEarfcnOrPci4GList" class="allowMoreInputParamsItemCls">
                                                <span>{{item}}</span>
                                                <span class="el-icon el-icon-close" @click="earfcnOrPci4GListDel(item)"></span>
                                            </div>
                                        </div>
                                        <el-form-item prop='pciLockEarfcnOrPci4GList' style="display:none;" label="" label-width="0px">
                                            <el-input v-model='ruleForm4G.pciLockEarfcnOrPci4GList'></el-input>
                                        </el-form-item>
                                    </div>
                                </div>
                            </div>
                            <div class="paramsItemBoxCls" style="margin-bottom: 5px;" v-show="ruleForm4G.scanMode == 'pcionlylock'">
                                <div class="paramsItemLabelCls">PCI</div>
                                <div class="allowMoreInputBoxCls">
                                    <div class="allowMoreInputContentCls">
                                        <div class="allowMoreInputFieldCls">
                                            <el-input v-model="ruleForm4G.pciOnlyLockPci4G"></el-input>
                                            <div class="allowMoreInputAddBtnCls" @click="pciOnlyLockPci4GAdd">
                                                <span class="el-icon el-icon-plus"></span>
                                                <span>Add</span>
                                            </div>
                                            <div class="allowMoreInputAddTipCls">
                                                <span style="color:red" v-if="pciErrorMessage4G">{{pciErrorMessage4G}}</span>
                                                <span style="color:rgba(0,0,0,0.32)" v-if="!pciErrorMessage4G"><%=rb.getString("FanWei")%>：0~503,Integer</span>
                                            </div>
                                        </div>
                                        <div class="allowMoreInputParamsCls">
                                            <div v-for="item in ruleForm4G.pciOnlyLockPci4GList" class="allowMoreInputParamsItemCls">
                                                <span>{{item}}</span>
                                                <span class="el-icon el-icon-close" @click="pci4GListDel(item)"></span>
                                            </div>
                                        </div>
                                        <el-form-item prop='pciOnlyLockPci4GList' style="display:none;" label="" label-width="0px">
                                            <el-input v-model='ruleForm4G.pciOnlyLockPci4GList'></el-input>
                                        </el-form-item>
                                    </div>
                                </div>
                            </div>
                        </el-form>
                    </div>
                </el-collapse-item>
                <el-collapse-item name="5GCPE" v-show="cpeModelType != '4G'">
                    <template slot='title'>
                        <p style="display:inline-block;margin-left:40px;">
                            <span style="font-size:14px;font-weight:bold">5G Frequency Lock</span>
                        </p>
                    </template>
                    <div class="rightContentCls">
                        <el-form ref="ruleForm5G" :model="ruleForm5G" :rules="rules4G" label-position="top">
                            <div class="paramsItemBoxCls">
                                <div class="paramsItemLabelCls"><%=rb.getString("SaoMiaoFangShi")%></div>
                                <el-form-item prop='nrLockMode' style="width:80%;min-width:400px;" label="">
                                    <el-radio-group v-model="ruleForm5G.nrLockMode" >
                                        <el-radio label="fullband" border size="small">Full Band</el-radio>
                                        <el-radio label="freqlock" border size="small">Frequency Lock</el-radio>
                                        <el-radio label="celllock" border size="small">Cell Lock</el-radio>
                                        <el-radio label="bandlock" border size="small">Band Lock</el-radio>
                                    </el-radio-group>
                                </el-form-item>
                            </div>
                            <div class="paramsItemBoxCls" style="margin-bottom: 5px;" v-show="ruleForm5G.nrLockMode == 'freqlock'">
                                <div class="paramsItemLabelCls">
                                    <div>Frequency Lock</div>
                                    <div class="allowMoreInputAddBtnCls" @click="add5GFreqClick">
                                        <span class="el-icon el-icon-plus"></span>
                                        <span>Add</span>
                                    </div>
                                    <div class="allowMoreInputAddTipCls">
                                        <span style="color:rgba(0,0,0,0.32)">5G NR：No more than 10,4G LTE：No more than 2</span>
                                    </div>
                                </div>
                                <div class="add5GCpePciLockBoxCls" v-show="add5GFreqShow">
                                    <div v-for="(item,index) in ruleForm5G.add5GFreqForm">
                                        <el-form ref="add5GFreqForm" :model='item' :rules='add5GFreqRules' label-position="top">     		     			            
                                            <el-form-item prop='rat' style="width:20%;min-width:210px;" label="Rat" label-width="160px" class="selectErrcCls">
                                                <el-select v-model='item.rat' @change="freqLockRatChange">
                                                    <el-option label="4G LTE" value="0"></el-option>
                                                    <el-option label="5G NR" value="1"></el-option>
                                                </el-select>
                                            </el-form-item>
                                            <el-form-item prop='band' style="width:20%;min-width:210px;" label="Band" label-width="160px" class="selectErrcCls">
                                                <el-select v-model='item.band' filterable>
                                                    <el-option v-for="item in bandList" :label="item?item:'Full'" :value="item"></el-option>
                                                </el-select>
                                            </el-form-item>
                                            <el-form-item prop='freq' style="width:40%;min-width:400px;" label="Freq" label-width="160px" class="validate-item">
                                                <el-input v-model.trim='item.freq'>
                                                    <template slot="append">{{add5GFreq_FreqRangeTips}}</template>
                                                </el-input>
                                            </el-form-item>
                                        </el-form>
                                    </div>
                                    <div class="add5GCpePciLockBtnBoxCls">
                                        <el-button type="primary" @click="add5GFreqSubmit"><%=rb.getString("QueDing")%></el-button>
                                        <el-button @click="add5GFreqClose"><%=rb.getString("QuXiao")%></el-button>
                                    </div>		
                                </div>
                                <div class="allowMoreInputBoxCls">
                                    <div v-for="(item,index) in ruleForm5G.freqLock5GList" class="allowMoreTableParamsItemCls">
                                        <div style="width: 30px;margin-left:15px;">{{index}}</div>
                                        <div style="width: 180px;">
                                            Rat: 
                                            <span v-show="item.rat == '0'">4G LTE</span>
                                            <span v-show="item.rat == '1'">5G NR</span>
                                        </div>
                                        <div style="width: 180px;">Band: {{item.band}}</div>
                                        <div style="width: 180px;">Freq: {{item.freq}}</div>
                                        <span class="el-icon el-icon-close" @click="frequencyLock5GListDel(index)"></span>
                                    </div>
                                </div>
                                <el-form-item prop='freqLock5GList' style="display:none;" label="" label-width="0px">
                                    <el-input v-model='ruleForm5G.freqLock5GList'></el-input>
                                </el-form-item>
                            </div>
                            <div class="paramsItemBoxCls" style="margin-bottom: 5px;" v-show="ruleForm5G.nrLockMode == 'celllock'">
                                <div class="paramsItemLabelCls">
                                    <div>Cell Lock</div>
                                    <div class="allowMoreInputAddBtnCls" @click="add5GCellLockClick">
                                        <span class="el-icon el-icon-plus"></span>
                                        <span>Add</span>
                                    </div>
                                </div>
                                <div class="add5GCpePciLockBoxCls" v-show="add5GCellLockShow">
                                    <div v-for="(item,index) in ruleForm5G.add5GCellLockForm">
                                        <el-form ref="add5GCellLockForm" :model='item' :rules='add5GCellLockRules' label-position="top">     		     			            
                                            <el-form-item prop='rat' style="width:20%;min-width:210px;" label="Rat" label-width="160px" class="selectErrcCls">
                                                <el-select v-model='item.rat'>
                                                    <el-option label="LTE" value="0"></el-option>
                                                    <el-option label="NR" value="1"></el-option>
                                                </el-select>
                                            </el-form-item>
                                            <el-form-item prop='band' style="width:20%;min-width:210px;" label="Band" label-width="160px" class="selectErrcCls">
                                                <el-select v-model='item.band' filterable>
                                                    <el-option v-for="item in bandList" :label="item?item:'Full'" :value="item"></el-option>
                                                </el-select>
                                            </el-form-item>
                                            <el-form-item prop='earfcn' style="width:20%;min-width:400px;" label="Earfcn" label-width="160px" class="validate-item">
                                                <el-input v-model.trim='item.earfcn'>
                                                    <template slot="append"><%=rb.getString("FanWei")%>：0~3279156,Integer</template>
                                                </el-input>
                                            </el-form-item>
                                            <el-form-item prop='pci' style="width:20%;min-width:400px;" label="PCI" label-width="160px" class="validate-item">
                                                <el-input v-model.trim='item.pci'>
                                                    <template slot="append"><%=rb.getString("FanWei")%>：0~1007,Integer</template>
                                                </el-input>
                                            </el-form-item>
                                        </el-form>
                                    </div>
                                    <div class="add5GCpePciLockBtnBoxCls">
                                        <el-button type="primary" @click="add5GCellLockSubmit"><%=rb.getString("QueDing")%></el-button>
                                        <el-button @click="add5GCellLockClose"><%=rb.getString("QuXiao")%></el-button>
                                    </div>		
                                </div>
                                <div class="allowMoreInputBoxCls">
                                    <div v-for="(item,index) in ruleForm5G.cellLock5GList" class="allowMoreTableParamsItemCls">
                                        <div style="width: 30px;margin-left:15px;">{{index}}</div>
                                        <div style="width: 180px;">
                                            Rat: 
                                            <span v-show="item.rat == '0'">LTE</span>
                                            <span v-show="item.rat == '1'">NR</span>
                                        </div>
                                        <div style="width: 180px;">Band: {{item.band}}</div>
                                        <div style="width: 180px;">Earfcn: {{item.earfcn}}</div>
                                        <div style="width: 180px;">PCI: {{item.pci}}</div>
                                        <span class="el-icon el-icon-close" @click="cellLock5GListDel(index)"></span>
                                    </div>
                                </div>
                                <el-form-item prop='cellLock5GList' style="display:none;" label="" label-width="0px">
                                    <el-input v-model='ruleForm5G.cellLock5GList'></el-input>
                                </el-form-item>
                            </div>
                            <div class="paramsItemBoxCls" style="margin-bottom: 5px;" v-show="ruleForm5G.nrLockMode == 'bandlock'">
                                <div class="paramsItemLabelCls">
                                    <div>Band Lock</div>
                                    <div class="allowMoreInputAddBtnCls" @click="add5GBandLockClick">
                                        <span class="el-icon el-icon-plus"></span>
                                        <span>Add</span>
                                    </div>
                                </div>
                                <div class="add5GCpePciLockBoxCls" v-show="add5GBandLockShow">
                                    <div v-for="(item,index) in ruleForm5G.add5GBandLockForm">
                                        <el-form ref="add5GBandLockForm" :model='item' :rules='add5GBandLockRules' label-position="top">     		     			            
                                            <el-form-item prop='rat' style="width:20%;min-width:210px;" label="Rat" label-width="160px" class="selectErrcCls">
                                                <el-select v-model='item.rat'>
                                                    <el-option label="LTE" value="0"></el-option>
                                                    <el-option label="NR" value="1"></el-option>
                                                </el-select>
                                            </el-form-item>
                                            <el-form-item prop='band' style="width:20%;min-width:210px;" label="Band" label-width="160px" class="selectErrcCls">
                                                <el-select v-model='item.band' filterable>
                                                    <el-option v-for="item in bandList" :label="item?item:'Full Band'" :value="item"></el-option>
                                                </el-select>
                                            </el-form-item>
                                        </el-form>
                                    </div>
                                    <div class="add5GCpePciLockBtnBoxCls">
                                        <el-button type="primary" @click="add5GBandLockSubmit"><%=rb.getString("QueDing")%></el-button>
                                        <el-button @click="add5GBandLockClose"><%=rb.getString("QuXiao")%></el-button>
                                    </div>		
                                </div>
                                <div class="allowMoreInputBoxCls">
                                    <div v-for="(item,index) in ruleForm5G.bandLock5GList" class="allowMoreTableParamsItemCls">
                                        <div style="width: 30px;margin-left:15px;">{{index}}</div>
                                        <div style="width: 180px;">
                                            Rat: 
                                            <span v-show="item.rat == '0'">LTE</span>
                                            <span v-show="item.rat == '1'">NR</span>
                                        </div>
                                        <div style="width: 180px;">Band: {{item.band?item.band:'Full Band'}}</div>
                                        <span class="el-icon el-icon-close" @click="bandLock5GListDel(index)"></span>
                                    </div>
                                </div>
                                <el-form-item prop='bandLock5GList' style="display:none;" label="" label-width="0px">
                                    <el-input v-model='ruleForm5G.bandLock5GList'></el-input>
                                </el-form-item>
                            </div>
                        </el-form>
                    </div>
                </el-collapse-item>
            </el-collapse>
		</div>
		<div class='itemMainBoxFooter'>
			<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var cpeLteOrNrPage = new Vue({
	el: '#cpeLteOrNrPage', 
	data() {
		var vm = this,
			validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

				if(value == '' || value == undefined || value == null){
					if(isRequired){
						callback(new Error(mag))
					}else{
						callback();
					}
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						callback(new Error(mag))
					}
				}
			},
            validateFreq = function(rule,value,callback) {
                var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/,
                    bandVal = vm.ruleForm5G.add5GFreqForm[0].band,
                    ratType = vm.ruleForm5G.add5GFreqForm[0].rat,
                    rangeStr = ratType == '0' ? vm.bandCounterpartFreq4GList[bandVal] : vm.bandCounterpartFreq5GList[bandVal],
                    minVal = parseInt(rangeStr.split('~')[0]),
                    maxVal = parseInt(rangeStr.split('~')[1]);

                if(value == '' || value == undefined || value == null) {
                    callback(new Error('error'))
                }else {
                    if(reg.test(value) && value >= minVal && value <= maxVal) {
                        callback();
                    }else {
                        callback('error');
                    }
                }
            },
            validateEarfcn = function(rule,value,callback) {
                if(value !== '') {
                    if(value - 0 < 0 || value - 3279156 > 0 || isNaN(value)) {
                        callback('<%=rb.getString("5GPinDianFanWei")%>');
                    }else {
                        callback();
                    }
                }else {
                    callback('Please Input Earfcn');
                }
            },
            validatePCI = function(rule,value,callback) {
                if(value !== '') {
                    if(value - 0 < 0 || value - 1007 > 0 || isNaN(value)) {
                        callback('<%=rb.getString("5GSpecificPCIFanWei")%>');
                    }else {
                        callback();
                    }
                }else {
                    callback('Please Input PCI');
                }
            };
		return {
			activeCollapse:['4GCPE','5GCPE'],
			cpeCode:'',
            cpeModelType:'',
			ruleForm4G:{
                scanMode:'fullband',
				earfcn4G:'',
                earfcn4GList:[],
                pciLockEarfcn4G:'',
                pciLockPci4G:'',
                pciLockEarfcnOrPci4GList:[],
                pciOnlyLockPci4G:'',
                pciOnlyLockPci4GList:[],
			},
			rules4G:{},
			earfcnErrorMessage4G:'',
            earfcnOrPciErrorMessage4G:'',
            pciErrorMessage4G:'',

            ruleForm5G:{
                nrLockMode:'fullband',
                add5GFreqForm:[{
                    rat:'0',
                    band:'1',
                    freq:''
                }],
                freqLock5GList:[],
                add5GCellLockForm:[{
                    rat:'0',
                    band:'1',
                    earfcn:'',
                    pci:''
                }],
                cellLock5GList:[],
                add5GBandLockForm:[{
                    rat:'0',
                    band:'1',
                }],
                bandLock5GList:[],
            },
            add5GFreqShow:false,
            add5GFreqRules:{
                freq: [{validator: validateFreq}],
            },
            bandCounterpartFreq4GList:{
                '1':'0~599','2':'600~1199','3':'1200~1949','4':'1950~2399','5':'2400~2649','7':'2750~3449','8':'3450~3799','12':'5010~5179',
                '13':'5180~5279','14':'5280~5379','17':'5730~5849','18':'5850~5999','19':'6000~6149','20':'6150~6449','25':'8040~8689',
                '26':'8690~9039','28':'9210~9659','29':'9660~9769','30':'9770~9869','32':'9920~10359','34':'36200~36349','38':'37750~38249',
                '39':'38250~38649','40':'38650~39649','41':'39650~41589','42':'41590~43589','43':'43590~45589','46':'46790~54539','48':'55240~56739',
                '66':'66436~67335','71':'68586~68935'
            },
            bandCounterpartFreq5GList:{
                '1':'422000~434000','2':'386000~398000','3':'361000~376000','5':'173800~178800','7':'524000~538000','8':'185000~192000',
                '12':'145800~149200','13':'149200~151200','14':'151600~153600','18':'172000~175000','20':'158200~164200','25':'386000~399000',
                '26':'171800~178800','28':'151600~160600','29':'65535~65535','30':'470000~472000','38':'514000~524000','40':'460000~480000',
                '41':'499200~537999','48':'636667~646666','66':'422000~440000','70':'399000~404000','71':'123400~130400','75':'286400~303400',
                '76':'285400~286400','77':'620000~680000','78':'620000~653333','79':'693334~733333'
            },
            add5GCellLockShow:false,
            add5GCellLockRules:{
                earfcn: [{validator: validateEarfcn}],
                pci: [{validator: validatePCI}],
            },

            add5GBandLockShow:false,
            add5GBandLockRules:{},

		};
	},
	computed: {
        bandList() {
            var vm = this;
                nrLockMode = vm.ruleForm5G.nrLockMode;
            let list = [];
            if(nrLockMode == 'celllock' || nrLockMode == 'bandlock'){
                for(let i=0;i<=100;i++) {
                    list.push(i);
                }
            }else if(nrLockMode == 'freqlock'){
                var ratType = vm.ruleForm5G.add5GFreqForm[0].rat;
                if(ratType == '0'){
                    list = ['1','2','3','4','5','7','8','12','13','14','17','18','19','20','25','26','28','29','30','32','34','38','39','40','41','42','43','46','48','66','71'];
                }else if(ratType == '1'){
                    list = ['1','2','3','5','7','8','12','13','14','18','20','25','26','28','29','30','38','40','41','48','66','70','71','75','76','77','78','79'];
                }
            }
            return list;
        },
        add5GFreq_FreqRangeTips(){
            var vm = this;
                bandVal = vm.ruleForm5G.add5GFreqForm[0].band,
                ratType = vm.ruleForm5G.add5GFreqForm[0].rat,
                str = '';
            if(vm.ruleForm5G.nrLockMode == 'freqlock'){
                if(ratType == '0'){
                    str = '<%=rb.getString("FanWei")%>：'+ vm.bandCounterpartFreq4GList[bandVal] +',Integer'
                }else if(ratType == '1'){
                    str = '<%=rb.getString("FanWei")%>：'+ vm.bandCounterpartFreq5GList[bandVal] +',Integer'
                }
            }
            return str
        }
    },
	methods: {
		init(itemParam,code,rowData){
			var vm = this;
			vm.cpeCode = code;
            vm.cpeModelType = rowData.cpe_model;
            if(vm.cpeModelType == '4G'){
                vm.get4GCpeParamData(code);
            }else{
                vm.get5GCpeParamData(code);
            }
		},
		get4GCpeParamData(code) {
			var vm = this,
				codes = [],
				url = '${ctx}/cell/CPE/getSettingParams.action',
				params = {
					cpeCode: code
				};
			axios.post(url, stringify(params)).then(function(res){
				var data = res.data,
                    earfcns = data["frequency"]?data["frequency"].split(','):[],
                    pciLockPci4Gs = data["PCI"]?data["PCI"].split(','):[],
                    pciOnlyLockPci4Gs = data["PCI_value"]?data["PCI_value"].split(','):[];

				vm.ruleForm4G.scanMode = data["lockMode"] ? data["lockMode"] : 'fullband';

                if(vm.ruleForm4G.scanMode == 'freqpreferred'){	
                    vm.ruleForm4G.earfcn4GList = earfcns.map(function(item){	
                        return item
                    });
                }else if(vm.ruleForm4G.scanMode == 'pcilock'){
                    vm.ruleForm4G.pciLockEarfcnOrPci4GList = earfcns.map(function(item,index){
                        return item + ':' + pciLockPci4Gs[index]
                    });
                }else if (vm.ruleForm4G.scanMode == 'pcionlylock'){
                    vm.ruleForm4G.pciOnlyLockPci4GList = pciOnlyLockPci4Gs.map(function(item){
                        return item
                    });
                }
                initForm(vm.$refs.ruleForm4G);
			});
		},
        get5GCpeParamData(code) {
			var vm = this,
				codes = [],
				url = '${ctx}/cell/CPE/getSettingParams.action',
				params = {
					cpeCode: code
				};
			axios.post(url, stringify(params)).then(function(res){
				var data = res.data,
                    freqLockList = data["nrFreq"]?data["nrFreq"].split(';'):[],
                    cellLockList = data["nrPci"]?data["nrPci"].split(';'):[],
                    bandLockList = data["nrBand"]?data["nrBand"].split(';'):[];

				vm.ruleForm5G.nrLockMode = data["nrLockMode"] ? data["nrLockMode"] : 'fullband';
                if(vm.ruleForm5G.nrLockMode== 'freqlock'){
                    vm.ruleForm5G.freqLock5GList = freqLockList.map(function(item,index){
                        let arr = item.split(',');
                        return {
                            rat:arr[0],
                            band:arr[1],
                            freq:arr[2],
                        }
                    });
                }else if(vm.ruleForm5G.nrLockMode == 'celllock'){	
                    vm.ruleForm5G.cellLock5GList = cellLockList.map(function(item){	
                        let arr = item.split(',');
                        return {
                            rat:arr[0],
                            band:arr[1],
                            earfcn:arr[2],
                            pci:arr[3]
                        }
                    });
                }else if (vm.ruleForm5G.nrLockMode == 'bandlock'){
                    vm.ruleForm5G.bandLock5GList = bandLockList.map(function(item){
                        let arr = item.split(',');
                        return {
                            rat:arr[0],
                            band:arr[1],
                        }
                    });
                }
                initForm(vm.$refs.ruleForm5G);
			});
		},
		// 4GCPE  scan Mode为Band/Frequency Preferred时  Earfcn添加事件
        earfcn4GAdd(){
            var vm = this,
                value = vm.ruleForm4G.earfcn4G.trim();

            if(value) {
                if(value - 0 < 0 || value - 65535 > 0) {
                    vm.earfcnErrorMessage4G = '<%=rb.getString("FanWei")%>：0~65535,Integer';
                }else {
                    if(vm.ruleForm4G.earfcn4GList.indexOf(value) == -1){
                        vm.ruleForm4G.earfcn4GList.push(value);
                        vm.ruleForm4G.earfcn4G = '';
                        vm.earfcnErrorMessage4G = '';
                    }else{
                        vm.earfcnErrorMessage4G = '<%=rb.getString("YiCunZai")%>';
                    }
                }
            }
		},
        // 4GCPE  scan Mode为Band/Frequency Preferred时  Earfcn 删除事件
        earfcnList4GDel(item){
            var vm = this;
			
            vm.ruleForm4G.earfcn4GList = vm.ruleForm4G.earfcn4GList.filter((items)=>{
                return items != item
            })
        },
        // 4GCPE  scan Mode为PCI lock时  Earfcn or pci添加事件
        earfcnOrPci4GAdd(){
            var vm = this,
                regNum = /^\d+$/,
                earfcn = vm.ruleForm4G.pciLockEarfcn4G.trim(),
                pci = vm.ruleForm4G.pciLockPci4G.trim();

            if(!regNum.test(earfcn) || ( earfcn - 0 < 0 || earfcn - 65535 > 0)) {
                vm.earfcnOrPciErrorMessage4G = 'Earfcn <%=rb.getString("FanWei")%>：0 ~65535, PCI <%=rb.getString("FanWei")%>：0~503';
            }else if(!regNum.test(pci) || (pci - 0 < 0 || pci-503 > 0)){
                vm.earfcnOrPciErrorMessage4G = 'Earfcn <%=rb.getString("FanWei")%>：0 ~65535, PCI <%=rb.getString("FanWei")%>：0~503';
            }else{
                let str = earfcn + ":" + pci;
                if( vm.ruleForm4G.pciLockEarfcnOrPci4GList.indexOf(str) == -1){
                    vm.ruleForm4G.pciLockEarfcnOrPci4GList.push(str);
                    vm.ruleForm4G.pciLockEarfcn4G = '';
                    vm.ruleForm4G.pciLockPci4G = '';
                    vm.earfcnOrPciErrorMessage4G = '';
                }else{
                    vm.earfcnOrPciErrorMessage4G = '<%=rb.getString("YiCunZai")%>';
                } 
            }
        },
        // 4GCPE  scan Mode为PCI lock时  Earfcn or pci 删除事件
        earfcnOrPci4GListDel(item){
            var vm = this;
			
            vm.ruleForm4G.pciLockEarfcnOrPci4GList = vm.ruleForm4G.pciLockEarfcnOrPci4GList.filter((items)=>{
                return items != item
            })
        },
         // 4GCPE  scan Mode为PCI only lock时  pci添加事件
        pciOnlyLockPci4GAdd(){
            var vm = this,
                value = vm.ruleForm4G.pciOnlyLockPci4G.trim();

            if(value) {
                if(value - 0 < 0 || value - 503 > 0) {
                    vm.pciErrorMessage4G = '<%=rb.getString("FanWei")%>：0~503,Integer';
                }else {
                    if(vm.ruleForm4G.pciOnlyLockPci4GList.indexOf(value) == -1){
                        vm.ruleForm4G.pciOnlyLockPci4GList.push(value);
                        vm.ruleForm4G.pciOnlyLockPci4G = '';
                        vm.pciErrorMessage4G = '';
                    }else{
                        vm.pciErrorMessage4G = '<%=rb.getString("YiCunZai")%>';
                    }
                }
            }
        },
         // 4GCPE  scan Mode为PCI only lock时  pci 删除事件
         pci4GListDel(item){
            var vm = this;
			
            vm.ruleForm4G.pciOnlyLockPci4GList = vm.ruleForm4G.pciOnlyLockPci4GList.filter((items)=>{
                return items != item
            })
        },
        // 5GCPE  scan Mode为freq lock时  点击新增
        add5GFreqClick(){
            var vm = this;

            vm.add5GFreqShow = true;
        },
        // 5GCPE  scan Mode为freq lock时  rat改变事件
        freqLockRatChange(){
            var vm = this;
            vm.ruleForm5G.add5GFreqForm[0].band = '1';
        },
        // 5GCPE  scan Mode为freq lock时  新增提交事件
        add5GFreqSubmit(){
            var vm = this;
            
            var freqLockList = vm.ruleForm5G.freqLock5GList;
            var nrList = [],lteList = [];
            freqLockList.map((item)=>{
                if(item.rat == '0'){
                    lteList.push(item)
                }else{
                    nrList.push(item)
                }
            })
            if((lteList.length == 2 && vm.ruleForm5G.add5GFreqForm[0].rat == '0') || (nrList.length == 10 && vm.ruleForm5G.add5GFreqForm[0].rat == '1')){
                vm.$message.warning('5G NR：No more than 10,4G LTE：No more than 2')
                return
            }
            vm.$refs.add5GFreqForm[0].validate(function(r){
                if(r) {
                    vm.ruleForm5G.freqLock5GList.push(Object.assign({},vm.ruleForm5G.add5GFreqForm[0]));
                    vm.add5GFreqClose();
                }
            });
        },
        // 5GCPE  scan Mode为freq lock时  取消新增事件
        add5GFreqClose(){
            var vm = this,
                params = {
                    rat:'0',
                    band:'1',
                    freq:''
                };
            Object.assign(vm.ruleForm5G.add5GFreqForm[0],params);
            vm.$refs.add5GFreqForm[0].clearValidate();
            vm.add5GFreqShow = false;
        },
        // 5GCPE  scan Mode为freq lock时 删除事件
        frequencyLock5GListDel(idx){
            var vm = this;
            vm.ruleForm5G.freqLock5GList.splice(idx,1);
        },
        // 5GCPE  scan Mode为 CellLock 时  点击新增
        add5GCellLockClick(){
            var vm = this;
            vm.add5GCellLockShow = true;
        },
        // 5GCPE  scan Mode为 CellLock 时  新增提交事件
        add5GCellLockSubmit(){
            var vm = this;

            vm.$refs.add5GCellLockForm[0].validate(function(r){
                if(r) {
                    vm.ruleForm5G.cellLock5GList.push(Object.assign({},vm.ruleForm5G.add5GCellLockForm[0]));
                    vm.add5GCellLockClose();
                }
            });
        },
        // 5GCPE  scan Mode为 CellLock 时  取消新增事件
        add5GCellLockClose(){
            var vm = this,
                params = {
                    rat:'0',
                    band:'1',
                    freq:''
                };
            Object.assign(vm.ruleForm5G.add5GCellLockForm[0],params);
            vm.$refs.add5GCellLockForm[0].clearValidate();
            vm.add5GCellLockShow = false;
        },
        // 5GCPE  scan Mode为 CellLock 时 删除事件
        cellLock5GListDel(idx){
            var vm = this;
            vm.ruleForm5G.cellLock5GList.splice(idx,1);
        },
         // 5GCPE  scan Mode为 BandLock 时  点击新增
         add5GBandLockClick(){
            var vm = this;
            vm.add5GBandLockShow = true;
        },
        // 5GCPE  scan Mode为 BandLock 时  新增提交事件
        add5GBandLockSubmit(){
            var vm = this;

            vm.$refs.add5GBandLockForm[0].validate(function(r){
                if(r) {
                    vm.ruleForm5G.bandLock5GList.push(Object.assign({},vm.ruleForm5G.add5GBandLockForm[0]));
                    vm.add5GBandLockClose();
                }
            });
        },
        // 5GCPE  scan Mode为 BandLock 时  取消新增事件
        add5GBandLockClose(){
            var vm = this,
                params = {
                    rat:'0',
                    band:'1',
                    freq:''
                };
            Object.assign(vm.ruleForm5G.add5GBandLockForm[0],params);
            vm.$refs.add5GBandLockForm[0].clearValidate();
            vm.add5GBandLockShow = false;
        },
        // 5GCPE  scan Mode为 BandLock 时 删除事件
        bandLock5GListDel(idx){
            var vm = this;
            vm.ruleForm5G.bandLock5GList.splice(idx,1);
        },
		// 判断是否为空
		isNull(val){
			if(val==undefined || val == null || val =="") return true;
			else return false;
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
		tableRowClassName({row,rowIndex}){
			if(row.operateType && row.operateType == 'remove'){
				return 'hidden-row'
			}
			return ''
		},
		settingsSubmit(){
			var vm = this;
            if(vm.cpeModelType == '4G'){
                vm.settings4GCpeSubmit();
            }else{
                vm.settings5GCpeSubmit();
            }
		},
        // 当设备类型为4G CPE时 提交设置参数
        settings4GCpeSubmit(){
            var vm = this,
                url = '${ctx}/cell/CPE/setCpeParams.action?',
			    params = {
					cpeCode:vm.cpeCode,
                    scanMode:vm.ruleForm4G.scanMode,
                    timeZone: timeZone,
                    pciChanged: 1,
                    selectModel:'4G'
				},
                isChanged = isFormChanged(vm.$refs.ruleForm4G);
            if(!isChanged){
                showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
                return;
            }
            if(vm.ruleForm4G.scanMode == 'freqpreferred'){
                if(vm.ruleForm4G.earfcn4GList.length == 0){
                    vm.earfcnErrorMessage4G = '<%=rb.getString("PinDianWeiKong")%>';
                    return
                }else{
                    vm.earfcnErrorMessage4G = '';
                    params.CPE_Frequency = vm.ruleForm4G.earfcn4GList.map(function(item){
                        return item;
                    }).join(',');					
                } 
            }else if(vm.ruleForm4G.scanMode == 'pcilock'){
                if(vm.ruleForm4G.pciLockEarfcnOrPci4GList.length == 0){
                    vm.earfcnOrPciErrorMessage4G = '<%=rb.getString("PinDianWeiKong")%> ';
                    return
                }else{
                    vm.earfcnOrPciErrorMessage4G = '';
                    var CPE_FrequencyList = [],
                        PCI_valueList = [];
                    vm.ruleForm4G.pciLockEarfcnOrPci4GList.map(function(item){
                        CPE_FrequencyList.push(item.split(':')[0]);
                        PCI_valueList.push(item.split(':')[1]);
                    });	
                    params.CPE_Frequency = CPE_FrequencyList.join(',');
                    params.PCI_value = PCI_valueList.join(',');
                }
            }else if(vm.ruleForm4G.scanMode == 'pcionlylock'){
                if(vm.ruleForm4G.pciOnlyLockPci4GList.length == 0){
                    vm.pciErrorMessage4G = '<%=rb.getString("PCIWeiKong")%>';
                    return
                }else{
                    vm.pciErrorMessage4G = '';	
                    params.PCI_value = vm.ruleForm4G.pciOnlyLockPci4GList.map(function(item){
                        return item;
                    }).join(',');					
                }
            }
			$('#cpeSettingOption').addClass('loading');
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type: 'success'
					});
					vm.closeSettings();
					$("#cpeSettingOption").removeClass("loading");
				}else{
					vm.$message({
						message: data["message"],
						type: 'error'
					});
				}
				$('#cpeSettingOption').removeClass('loading');
			});
        },
        // 当设备类型为5G CPE时 提交设置参数
        settings5GCpeSubmit(){
            var vm = this,
                url = '${ctx}/cell/CPE/setCpeParams.action',
			    params = {
					cpeCode:vm.cpeCode,
                    nrLockMode:vm.ruleForm5G.nrLockMode,
                    timeZone: timeZone,
                    pciChanged: 1,
                    selectModel:'5G'
				},
                isChanged = isFormChanged(vm.$refs.ruleForm5G);
            if(!isChanged){
                showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
                return;
            }
            if(vm.ruleForm5G.nrLockMode == 'freqlock'){
                if(vm.ruleForm5G.freqLock5GList.length == 0){
                    params.nrFreq = '';
                }else{
                    params.nrFreq = vm.ruleForm5G.freqLock5GList.map(function(item){
                        return item.rat +','+ item.band +','+ item.freq ;
                    }).join(';');			
                } 
            }else if(vm.ruleForm5G.nrLockMode == 'celllock'){
                if(vm.ruleForm5G.cellLock5GList.length == 0){
                    params.nrPci = '';
                }else{
                    params.nrPci = vm.ruleForm5G.cellLock5GList.map(function(item){
                        return item.rat +','+ item.band +','+ item.earfcn +','+ item.pci;
                    }).join(';');
                }
            }else if(vm.ruleForm5G.nrLockMode == 'bandlock'){
                if(vm.ruleForm5G.bandLock5GList.length == 0){
                    params.nrBand = '';
                }else{
                    params.nrBand = vm.ruleForm5G.bandLock5GList.map(function(item){
                        return item.rat +','+ item.band ;
                    }).join(';');				
                }
            }
			$('#cpeSettingOption').addClass('loading');
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type: 'success'
					});
					vm.closeSettings();
					$("#cpeSettingOption").removeClass("loading");
				}else{
					vm.$message({
						message: data["message"],
						type: 'error'
					});
				}
				$('#cpeSettingOption').removeClass('loading');
			});
        },
		closeSettings(){
			eventBus.$emit('close-cpe-setting');
		},
		isValidMacAddress(mac){
			var reg = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/;
			return reg.test(mac); 
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
        //校验子网掩码
        isMask(str){
            var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
            return exp.test(str); 		
        },
        // 验证输入的是否是数字
        isNumeric(str) {
            if(str.length==0){
                return false;
            }
            for(var i=0;i<str.length;i++){
                if(str.charAt(i)<"0" || str.charAt(i)>"9"){
                    return false;
                }
            }
            return true;  
        },
	},
	mounted() {
		eventBus.$off("cpe-data").$on("cpe-data",this.init)
	}
});

</script>
